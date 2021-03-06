package terraform

import (
	"bufio"
	"bytes"
	"fmt"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"log"
	"os"
	"os/exec"
	"strings"
)

type Inputs struct {
	Repository string
	TerraformPath string
	Environment string
	GitUser string
	GitToken string
	AwsAccessKeyId string
	AwsSecretAccessKey string
}

const commonsVar  = "-var-file=./variables/common.tfvars"

func Run(inputs Inputs) {
	log.Println("Terraform starting...")
	execCommand("terraform", "version")
	split := strings.Split(inputs.Repository, "/")
	dirRepo := strings.Replace(split[len(split)-1], ".git", "", -1)
	pwd , _ := os.Getwd()
	split = strings.Split(pwd, "/")
	pwd = split[len(split)-1]
	if pwd == dirRepo {
		err := inputs.pullRepo("")
		if err != nil {
			log.Fatal("Failed Pull repository. Error: ", err)
		}
		os.Chdir(inputs.TerraformPath)
	} else {
		log.Println("Cloning repository...")
		err := inputs.plainClone(dirRepo)
		if err != nil {
			log.Fatal("Failed cloning repository. Error: ", err)
		}
		os.Chdir(fmt.Sprint(dirRepo, "/", inputs.TerraformPath))
	}

	varFile := fmt.Sprintf("-var-file=variables/%v.tfvars", inputs.Environment)
	backendConfig := fmt.Sprintf("-backend-config=%v.tfbackend", inputs.Environment)
	//terraform init -var-file=./variables/common.tfvars -var-file=$(VARS) -reconfigure -backend-config=$(ENV).tfbackend
	execCommand("terraform", "init", commonsVar, varFile, "-reconfigure", backendConfig)
	//terraform plan -var-file=./variables/common.tfvars -var-file=$(VARS)
	execCommand("terraform", "plan", commonsVar, varFile)
	//terraform apply -var-file=./variables/common.tfvars -var-file=$(VARS) -auto-approve
	execCommand("terraform", "apply", commonsVar, varFile, "-auto-approve")
	log.Println("Terraform finished!!!!!!")
}

func (in Inputs) plainClone(dirRepo string) error {
	_, err := git.PlainClone(dirRepo, false, &git.CloneOptions{
		Auth: &http.BasicAuth{
			Username: in.GitUser,
			Password: in.GitToken,
		},
		URL:      in.Repository,
		Progress: os.Stdout,
	})
	if err != nil {
		if "repository already exists" != err.Error() {
			log.Fatal(err)
		}
		repo, _ := git.PlainOpen(dirRepo)
		w, _ := repo.Worktree()
		err = w.Pull(&git.PullOptions{
			RemoteName: "origin",
			Auth: &http.BasicAuth{
				Username: in.GitUser,
				Password: in.GitToken,
			},
			Progress: os.Stdout,
		})
		if "already up-to-date" == err.Error() {
			return nil
		}
	}
	return err
}

func (in Inputs) pullRepo(dirRepo string) error {
	log.Println("Pull repository...")
	repo, _ := git.PlainOpen(dirRepo)
	w, _ := repo.Worktree()
	err := w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth: &http.BasicAuth{
			Username: in.GitUser,
			Password: in.GitToken,
		},
		Progress: os.Stdout,
	})
	if "already up-to-date" == err.Error() {
		return nil
	}
	return err
}

func execCommand(command string, params ...string) {
	log.Printf("Executing command: %v params: %v\n", command, params)
	cmd := exec.Command(command, params...)
	stdout, _ := cmd.StdoutPipe()
	var outError bytes.Buffer
	cmd.Stderr = &outError
	cmd.Start()
	scanner := bufio.NewScanner(stdout)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
	err := cmd.Wait()
	if err != nil {
		log.Fatalf("Failed to execute command %v\nParams: %v\nError: %v", command, params, outError.String())
	}
}