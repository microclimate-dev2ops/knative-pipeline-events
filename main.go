package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	gh "gopkg.in/go-playground/webhooks.v3/github"
)

// Manual subsmission data struct
type BuildRequest struct {
	//Example Paylod
	//{
	//	"repourl": "https://github.ibm.com/duanes-org/slim-devops-test-overrides-project",
	//	"commitid": "7d84981c66718ee2dda1af280f915cc2feb9d275",
	//	"reponame": "slim-devops-test-overrides-project"
	//
	//}

	REPOURL  string `json:"repourl"`
	COMMITID string `json:"commitid"`
	REPONAME string `json:"reponame"`
}

func handleManualBuildRequest(w http.ResponseWriter, r *http.Request) {

	// fmt.Printf("headers: %v\n", r.Header)
	// ToDo: check the CE-X-CE TYPE header to know wht type of gh Payload it will be

	requestData := BuildRequest{}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		log.Println("OH CACK - GONE WRONG")
		return
	}

	dateTime := time.Now().Unix()

	log.Println("============================")
	log.Println(requestData.REPOURL)
	log.Println(requestData.COMMITID)
	log.Println(requestData.REPONAME)
	log.Println(dateTime)
	log.Println("============================")

	argmap := map[string]interface{}{
		"URL":     requestData.REPOURL,
		"SHORTID": requestData.COMMITID[0:7],
		"ID":      requestData.COMMITID,
		"NAME":    requestData.REPONAME,
		"MARKER":  dateTime,
	}

	submitBuild(argmap)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {

	gitHubEventType := r.Header["Ce-X-Ce-X-Github-Event"]

	gitHubEventTypeString := strings.Replace(gitHubEventType[0], "\"", "", -1)

	log.Println(gitHubEventTypeString)

	var argmap map[string]interface{}

	if gitHubEventTypeString == "push" {

		webhookData := gh.PushPayload{}
		err := json.NewDecoder(r.Body).Decode(&webhookData)
		if err != nil {
			log.Println("Something went wrong, webhook data was not decoded correctly")
			return
		}

		dateTime := time.Now().Unix()

		log.Println("============================")
		log.Println(webhookData.HeadCommit.ID)
		log.Println(webhookData.Repository.URL)
		log.Println(webhookData.Repository.Name)
		log.Println("============================")

		argmap = map[string]interface{}{
			"URL":     webhookData.Repository.URL,
			"SHORTID": webhookData.HeadCommit.ID[0:7],
			"ID":      webhookData.HeadCommit.ID,
			"NAME":    webhookData.Repository.Name,
			"MARKER":  dateTime,
		}

	} else if gitHubEventTypeString == "pull_request" {

		webhookData := gh.PullRequestPayload{}
		err := json.NewDecoder(r.Body).Decode(&webhookData)
		if err != nil {
			log.Println("Something went wrong, webhook data was not decoded correctly")
			return
		}

		dateTime := time.Now().Unix()

		log.Println("============================")
		log.Println(webhookData.PullRequest.Head.Sha)
		log.Println(webhookData.Repository.HTMLURL)
		log.Println(webhookData.Repository.Name)
		log.Println("============================")

		argmap = map[string]interface{}{
			"URL":     webhookData.Repository.HTMLURL,
			"SHORTID": webhookData.PullRequest.Head.Sha[0:7],
			"ID":      webhookData.PullRequest.Head.Sha,
			"NAME":    webhookData.Repository.Name,
			"MARKER":  dateTime,
		}
	}

	submitBuild(argmap)

}

func modifyYaml(gitAttrs map[string]interface{}, templateToChange, templateOutputFile string) (string, error) {
	templateFile, err := filepath.Abs(templateToChange)
	if err != nil {
		log.Printf("Error 1 : %s", err)
	}

	resourceyaml, err := ioutil.ReadFile(templateFile)
	if err != nil {
		log.Printf("Error 2 : %s", err)
	}

	// Can be any name really
	editedyaml := template.New(templateOutputFile)
	editedyaml, err = editedyaml.Parse(string(resourceyaml))
	if err != nil {
		log.Printf("Error 3 : %s", err)
	}

	var yml bytes.Buffer
	// Applies what's in the config string INTO the xml: effectively doing variable substitution.
	editedyaml.Execute(&yml, gitAttrs)
	data := yml.Bytes()

	err = ioutil.WriteFile("/tmp/"+templateOutputFile, data, 0644)
	if err != nil {
		log.Println("Error writing file")
	}

	return string(data[:]), nil

}

func submitBuild(varmap map[string]interface{}) {

	configString, err := modifyYaml(varmap, "templates/resource.yaml", "edited-resource.yaml")
	if err != nil {
		log.Println("WA WA WAAAAAA")
	}

	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	configString, err = modifyYaml(varmap, "templates/pipeline-run.yaml", "edited-pipeline-run.yaml")
	if err != nil {
		log.Println("WA WA WAAAAAA 2")
	}

	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	output, err := applyYaml("/tmp/edited-resource.yaml")
	if output != nil {
		log.Printf("%s", output)
	}
	if err != nil {
		log.Println("UH OH!!")
		return
	}

	output, err = applyYaml("/tmp/edited-pipeline-run.yaml")
	if output != nil {
		log.Printf("%s", output)
	}
	if err != nil {
		log.Println("UH OH!!")
		return
	}

	//kube config stuff
	/*	var cfg *rest.Config
		kubeconfig := os.Getenv("KUBECONFIG")
		if len(kubeconfig) != 0 {
			cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		} else {
			cfg, err = rest.InClusterConfig()
		}
		if err != nil {
			log.Printf("Error building kubeconfig from %s: %s", kubeconfig, err.Error())
		}

		//exec kubectl apply struff here
		pipeClient, err := pipe.NewForConfig(cfg)
		if err != nil {
			log.Printf("Error building kubernetes client: %s", err.Error())
		}

		pipeClient. */

}

func applyYaml(filename string) ([]byte, error) {
	//env := os.Environ()
	var commandArgs string
	commandArgs += "apply" + " " + "-f" + " " + filename
	splits := strings.Split(commandArgs, " ")
	log.Printf("Issuing Kubectl command with arguments `%s`", splits)
	cmd := exec.Command("kubectl", splits...)
	//cmd.Env = env
	return cmd.CombinedOutput()
}

func main() {
	log.Println("knative-devops-runtime server started")
	http.HandleFunc("/", handleWebhook)
	http.HandleFunc("/manual", handleManualBuildRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
