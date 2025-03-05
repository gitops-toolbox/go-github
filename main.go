package main

import (
	"fmt"

	"github.com/gitops-toolbox/go-github/config"
)

func main() {
	fmt.Println(config.Init())
	// g, err := org.GetClient("gitops-toolbox")
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// repo := org.Repo{
	// 	Name: "test",
	// 	Org:  g,
	// }
	// if repo.HasBranch("test") {
	// 	fmt.Println("Branch exists")
	// 	fmt.Println(repo.FetchBranchesByPrefix())
	// } else {
	// 	fmt.Println("Branch does not exist")
	// 	repo.UpdatePR("test", "main")
	// }
}
