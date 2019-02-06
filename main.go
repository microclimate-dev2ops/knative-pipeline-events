package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	gh "gopkg.in/go-playground/webhooks.v3/github"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"
)

//BuildInformation - good stuff
type BuildInformation struct {
	URL     string
	SHORTID string
	ID      string
	NAME    string
	MARKER  string
}

//BuildRequest - a manual submission data struct
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
	requestData := BuildRequest{}
	err := json.NewDecoder(r.Body).Decode(&requestData)
	if err != nil {
		// TODO we don't have errorf or warnf, would be useful to go with Logrus instead
		log.Printf("An error occurred decoding the manual build request body: %s", err)
		return
	}

	timestamp := getDateTimeAsString()

	log.Println("Printing request information...")
	log.Println("============================")
	log.Println(requestData.REPOURL)
	log.Println(requestData.COMMITID)
	log.Println(requestData.REPONAME)
	log.Println(requestData.BRANCH)
	log.Println(timestamp)
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

	buildInformation := BuildInformation{}
	buildInformation.URL = requestData.REPOURL
	buildInformation.SHORTID = shortid
	buildInformation.ID = id
	buildInformation.NAME = requestData.REPONAME
	buildInformation.MARKER = timestamp
	submitBuild(buildInformation)
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	gitHubEventType := r.Header["Ce-X-Ce-X-Github-Event"]
	gitHubEventTypeString := strings.Replace(gitHubEventType[0], "\"", "", -1)
	log.Printf("GitHub event type is %s", gitHubEventTypeString)

	buildInformation := BuildInformation{}

	if gitHubEventTypeString == "push" {
		log.Println("Handling push event")
		webhookData := gh.PushPayload{}
		err := json.NewDecoder(r.Body).Decode(&webhookData)
		if err != nil {
			log.Printf("An error occurred decoding webhook data: %s", err)
			return
		}
		// TODO have timestamp be automatically created when we create the struct (e.g. a constructor method?)
		timestamp := strconv.FormatInt(time.Now().Unix(), 10)
		buildInformation.URL = webhookData.Repository.URL
		buildInformation.SHORTID = webhookData.HeadCommit.ID[0:7]
		buildInformation.ID = webhookData.HeadCommit.ID
		buildInformation.NAME = webhookData.Repository.Name
		buildInformation.MARKER = timestamp

	} else if gitHubEventTypeString == "pull_request" {
		log.Println("Handling pull request event")
		webhookData := gh.PullRequestPayload{}
		err := json.NewDecoder(r.Body).Decode(&webhookData)
		if err != nil {
			log.Printf("An error occurred decoding webhook data: %s", err)
			return
		}

		buildInformation.URL = webhookData.Repository.HTMLURL
		buildInformation.SHORTID = webhookData.PullRequest.Head.Sha[0:7]
		buildInformation.ID = webhookData.PullRequest.Head.Sha
		buildInformation.NAME = webhookData.Repository.Name
		buildInformation.MARKER = getDateTimeAsString()
	}

	log.Printf("Build information: \n %s", buildInformation)

	submitBuild(buildInformation)
}

func getDateTimeAsString() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func modifyYaml(gitAttrs BuildInformation, templateToChange, templateOutputFile string) (string, error) {
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

func submitBuild(buildInformation BuildInformation) {
	resourceFileLocation := "templates/resource.yaml"
	pipelineRunFileLocation := "templates/pipeline-run.yaml"

	editedResourceFileOutputName := "edited-resource.yaml"
	editedResourceFileOutputFullPath := fmt.Sprintf("/tmp/%s", editedResourceFileOutputName)

	editedPipelineFileOutputName := "edited-pipeline-run.yaml"
	editedPipelineFileOutputFullPath := fmt.Sprintf("/tmp/%s", editedPipelineFileOutputName)

	configString, err := modifyYaml(buildInformation, resourceFileLocation, editedResourceFileOutputName)
	if err != nil {
		log.Printf("An error occurred modifying %s: %s", resourceFileLocation, err)
		return
	}

	log.Println("Printing the configuration string to use (stage one)")
	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	configString, err = modifyYaml(buildInformation, pipelineRunFileLocation, editedPipelineFileOutputName)
	if err != nil {
		log.Printf("An error occurred modifying %s: %s", pipelineRunFileLocation, err)
		return
	}

	log.Println("Printing the configuration string to use (stage two)")
	log.Println("============================")
	log.Println(configString)
	log.Println("============================")

	output, err := applyYaml(editedResourceFileOutputFullPath)
	if output != nil {
		log.Printf("Applied %s: \n %s", editedResourceFileOutputFullPath, output)
	}
	if err != nil {
		log.Printf("An error occurred applying the yaml at %s", editedResourceFileOutputFullPath)
		return
	}

	output, err = applyYaml(editedPipelineFileOutputFullPath)
	if output != nil {
		log.Printf("Applied %s: \n %s", editedPipelineFileOutputFullPath, output)
	}
	if err != nil {
		log.Printf("An error occurred applying the yaml at %s", editedPipelineFileOutputFullPath)
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
