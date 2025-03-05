package org

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v68/github"
	"golang.org/x/crypto/openpgp"
)

const BRANCH_PREFIX = "gitops-toolbox/"

type GitFile struct {
	Path    string
	Content []byte
	Mode    *string
}

type Repo struct {
	Name string
	Org  Org
}

// getRef returns the commit branch reference object if it exists or creates it
// from the base branch before returning it.
func (repo Repo) getRef(commitBranch, baseBranch string) (ref *github.Reference, err error) {
	ctx := repo.Org.ctx
	sourceOwner := repo.Org.Organization
	if ref, _, err = repo.Org.client.Git.GetRef(ctx, sourceOwner, repo.Name, "refs/heads/"+commitBranch); err == nil {
		return ref, nil
	}

	// We consider that an error means the branch has not been found and needs to
	// be created.
	if commitBranch == baseBranch {
		return nil, errors.New("the commit branch does not exist but base branch is the same as commit branch")
	}

	if baseBranch == "" {
		return nil, errors.New("the base branch should not be set to an empty string when the branch specified by commit branch does not exists")
	}

	var baseRef *github.Reference
	fmt.Println("Getting baseRef")
	if baseRef, _, err = repo.Org.client.Git.GetRef(ctx, sourceOwner, repo.Name, "refs/heads/"+baseBranch); err != nil {
		return nil, err
	}
	newRef := &github.Reference{Ref: github.Ptr("refs/heads/" + commitBranch), Object: &github.GitObject{SHA: baseRef.Object.SHA}}
	fmt.Printf("Creating a new reference %v\n", newRef)
	ref, _, err = repo.Org.client.Git.CreateRef(ctx, sourceOwner, repo.Name, newRef)
	return ref, err
}

// getTree generates the tree to commit based on the given files and the commit
// of the ref you got in getRef.
func (repo Repo) getTree(ref *github.Reference, gitFiles []GitFile) (tree *github.Tree, err error) {
	// Create a tree with what to commit.
	entries := []*github.TreeEntry{}

	// Load each file into the tree.
	for _, fileArg := range gitFiles {
		mode := fileArg.Mode
		if mode == nil {
			mode = github.Ptr("100644")
		}
		if fileArg.Content == nil {
			entries = append(entries, &github.TreeEntry{Path: github.Ptr(fileArg.Path), Content: nil, Mode: mode, SHA: nil})
			continue
		}
		entries = append(entries, &github.TreeEntry{Path: github.Ptr(fileArg.Path), Type: github.Ptr("blob"), Content: github.Ptr(string(fileArg.Content)), Mode: mode})
	}

	tree, _, err = repo.Org.client.Git.CreateTree(repo.Org.ctx, repo.Org.Organization, repo.Name, *ref.Object.SHA, entries)
	return tree, err
}

// pushCommit creates the commit in the given reference using the given tree.
func (repo Repo) pushCommit(message string, ref *github.Reference, tree *github.Tree) (err error) {
	// Get the parent commit to attach the commit to.
	parent, _, err := repo.Org.client.Repositories.GetCommit(repo.Org.ctx, repo.Org.Organization, repo.Name, *ref.Object.SHA, nil)
	if err != nil {
		return err
	}
	// This is not always populated, but is needed.
	parent.Commit.SHA = parent.SHA

	// Create the commit using the tree.
	date := time.Now()
	author := &github.CommitAuthor{Date: &github.Timestamp{Time: date}, Name: github.Ptr("GitopsToolbox"), Email: github.Ptr("gitopstoolbox@lanziani.com")}
	commit := &github.Commit{Author: author, Message: github.Ptr(message), Tree: tree, Parents: []*github.Commit{parent.Commit}}
	opts := github.CreateCommitOptions{}

	if repo.Org.privateKey != nil {
		armoredBlock, e := os.ReadFile(*repo.Org.privateKey)
		if e != nil {
			return e
		}
		keyring, e := openpgp.ReadArmoredKeyRing(bytes.NewReader(armoredBlock))
		if e != nil {
			return e
		}
		if len(keyring) != 1 {
			return errors.New("expected exactly one key in the keyring")
		}
		key := keyring[0]
		opts.Signer = github.MessageSignerFunc(func(w io.Writer, r io.Reader) error {
			return openpgp.ArmoredDetachSign(w, key, r, nil)
		})
	}

	newCommit, _, err := repo.Org.client.Git.CreateCommit(repo.Org.ctx, repo.Org.Organization, repo.Name, commit, &opts)
	if err != nil {
		return err
	}

	// Attach the commit to the master branch.
	ref.Object.SHA = newCommit.SHA
	_, _, err = repo.Org.client.Git.UpdateRef(repo.Org.ctx, repo.Org.Organization, repo.Name, ref, false)
	return err
}

// createPR creates a pull request. Based on: https://pkg.go.dev/github.com/google/go-github/github#example-PullRequestsService-Create
func (repo Repo) createPR(subject, commitBranch, description, baseBranch string) (err error) {

	newPR := &github.NewPullRequest{
		Title:               github.Ptr(subject),
		Head:                github.Ptr(commitBranch),
		HeadRepo:            github.Ptr(repo.Name),
		Base:                github.Ptr(baseBranch),
		Body:                github.Ptr(description),
		MaintainerCanModify: github.Ptr(true),
	}

	pr, _, err := repo.Org.client.PullRequests.Create(repo.Org.ctx, repo.Org.Organization, repo.Name, newPR)
	if err != nil {
		return err
	}

	fmt.Printf("PR created: %s\n", pr.GetHTMLURL())
	return nil
}

func (repo Repo) FetchBranchesByPrefix() ([]string, error) {
	ctx := repo.Org.ctx
	sourceOwner := repo.Org.Organization
	result := []string{}

	branches, _, err := repo.Org.client.Repositories.ListBranches(ctx, sourceOwner, repo.Name, nil)
	if err != nil {
		return nil, err
	}

	for _, branch := range branches {
		name := branch.GetName()
		if strings.HasPrefix(name, BRANCH_PREFIX) {
			result = append(result, branch.GetName())
		}
	}

	return result, nil
}

func (repo Repo) HasBranch(commitBranch string) bool {
	ctx := repo.Org.ctx
	sourceOwner := repo.Org.Organization
	commitBranch = BRANCH_PREFIX + commitBranch

	_, _, err := repo.Org.client.Repositories.GetBranch(ctx, sourceOwner, repo.Name, commitBranch, 10)
	return err == nil
}

func (repo Repo) UpdatePR(newBranch, baseBranch string) {
	commitBranch := BRANCH_PREFIX + newBranch

	GitFiles := []GitFile{
		{"github-interface.md", nil, nil},
	}

	ref, err := repo.getRef(commitBranch, baseBranch)
	if err != nil {
		log.Fatalf("Unable to get/create the commit reference: %s\n", err)
	}
	if ref == nil {
		log.Fatalf("No error where returned but the reference is nil")
	}

	tree, err := repo.getTree(ref, GitFiles)
	if err != nil {
		log.Fatalf("Unable to create the tree based on the provided files: %s\n", err)
	}

	if err := repo.pushCommit("my commit", ref, tree); err != nil {
		log.Fatalf("Unable to create the commit: %s\n", err)
	}

	if err := repo.createPR("Subject", commitBranch, "This is an automated PR", baseBranch); err != nil {
		log.Fatalf("Error while creating the pull request: %s", err)
	}
}
