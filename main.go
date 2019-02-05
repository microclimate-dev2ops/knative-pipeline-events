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
	/* Example payload
	{
	  "repourl": "https://github.ibm.com/duanes-org/slim-devops-test-overrides-project",
	  "commitid": "7d84981c66718ee2dda1af280f915cc2feb9d275",
	  "reponame": "slim-devops-test-overrides-project"
	}
	*/
	REPOURL  string `json:"repourl"`
	COMMITID string `json:"commitid"`
	REPONAME string `json:"reponame"`
	BRANCH   string `json:"branch"`
}

func handleManualBuildRequest(w http.ResponseWriter, r *http.Request) {
	// fmt.Printf("headers: %v\n", r.Header)
	// TODO: check the CE-X-CE TYPE header to know what type of GitHub Payload it will be
	requestData := BuildRequest{}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		// TODO we don't have errorf or warnf, would be useful to go with Logrus instead
		log.Printf("An error occurred decoding the manual build request body: %s", err)
		return
	}

	dateTime := time.Now().Unix()

	log.Println("Printing request information...")
	log.Println("============================")
	log.Println(requestData.REPOURL)
	log.Println(requestData.COMMITID)
	log.Println(requestData.REPONAME)
	log.Println(requestData.BRANCH)
	log.Println(dateTime)
	log.Println("============================")

	id := ""
	shortid := ""
	if requestData.COMMITID != "" {
		id = requestData.COMMITID
		shortid = requestData.COMMITID[0:7]
	} else {
		id = requestData.BRANCH
		shortid = "latest"
	}

	argmap := map[string]interface{}{
		"URL":     requestData.REPOURL,
		"SHORTID": shortid,
		"ID":      id,
		"NAME":    requestData.REPONAME,
		"MARKER":  dateTime,
	}
	submitBuild(argmap)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	gitHubEventType := r.Header["Ce-X-Ce-X-Github-Event"]
	gitHubEventTypeString := strings.Replace(gitHubEventType[0], "\"", "", -1)
	log.Printf("GitHub event type is %s", gitHubEventTypeString)

	var argmap map[string]interface{}

	if gitHubEventTypeString == "push" {
		webhookData := gh.PushPayload{}
		err := json.NewDecoder(r.Body).Decode(&webhookData)
		if err != nil {
			log.Printf("An error occurred decoding webhook data: %s", err)
			return
		}

		dateTime := time.Now().Unix()
		log.Println("Printing webhook information for a push event...")
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
			log.Printf("An error occurred decoding webhook data: %s", err)
			return
		}

		dateTime := time.Now().Unix()

		log.Println("Printing webhook information for a pull request...")
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
		log.Printf("An error occurred getting the path of the template file to change: %s", err)
		return "", err
	}

	resourceyaml, err := ioutil.ReadFile(templateFile)
	if err != nil {
		log.Printf("An error occurred reading the template file to change: %s", err)
		return "", err
	}

	// Can be any name really
	editedyaml := template.New(templateOutputFile)
	editedyaml, err = editedyaml.Parse(string(resourceyaml))
	if err != nil {
		log.Printf("An error occurred parsing the resource yaml: %s", err)
		return "", err
	}

	var yml bytes.Buffer
	// Applies what's in the config string INTO the xml: effectively doing variable substitution.
	editedyaml.Execute(&yml, gitAttrs)
	data := yml.Bytes()

	location := "/tmp/" + templateOutputFile
	err = ioutil.WriteFile(location, data, 0644)
	if err != nil {
		log.Printf("Error writing file to %s", location)
	}
	return string(data[:]), nil
}

func submitBuild(varmap map[string]interface{}) {
	configString, err := modifyYaml(varmap, "templates/resource.yaml", "edited-resource.yaml")
	if err != nil {
		log.Printf("An error occurred modifying templates/resource.yaml: %s", err)
		return
	}

	log.Println("Printing the configuration string to use (stage one)")
	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	configString, err = modifyYaml(varmap, "templates/pipeline-run.yaml", "edited-pipeline-run.yaml")
	if err != nil {
		log.Printf("An error occurred modifying templates/pipeline-run.yaml: %s", err)
		return
	}

	log.Println("Printing the configuration string to use (stage two)")
	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	output, err := applyYaml("/tmp/edited-resource.yaml")
	if output != nil {
		log.Printf("Applied /tmp/edited-resource.yaml: %s", output)
	}
	if err != nil {
		log.Printf("An error occurred applying the yaml at \n /tmp/edited-resource.yaml")
		return
	}

	output, err = applyYaml("/tmp/edited-pipeline-run.yaml")
	if output != nil {
		log.Printf("Applied /tmp/edited-pipeline-run.yaml: \n %s", output)
	}
	if err != nil {
		log.Printf("An error occurred applying the yaml at /tmp/edited-pipeline-run.yaml")
		return
	}
}

func applyYaml(filename string) ([]byte, error) {
	var commandArgs string
	commandArgs += "apply" + " " + "-f" + " " + filename
	splits := strings.Split(commandArgs, " ")
	log.Printf("Issuing Kubectl command with arguments `%s`", splits)
	cmd := exec.Command("kubectl", splits...)
	return cmd.CombinedOutput()
}

func main() {
	log.Println("knative-devops-runtime server started")
	http.HandleFunc("/", handleWebhook)
	http.HandleFunc("/manual", handleManualBuildRequest)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
