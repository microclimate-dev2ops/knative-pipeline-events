# knative-pipeline-events

## Prerequisites

- Docker for Mac - switch to edge version
    - Under advanced set CPU 6, Memory 10, Swap 2.5
    - Under Daemon add insecure registry - `host.docker.internal:5000`
    - Enable Kubernetes
    - A properly set up `GOPATH` (it is advised to use `$HOME/go`). The directory structure is important so that `ko` commands function as expected. Images should be built and made available at your `localhost:5000` Docker registry when `ko` is used. See [Gopath docs](https://github.com/golang/go/wiki/GOPATH) for details.

- Set up a local docker registry  and optionally a registry viewer

  ```
  docker run -d --rm -p 5000:5000 --name registry-srv -e REGISTRY_STORAGE_DELETE_ENABLED=true registry:2

  docker run -d --rm -p 8080:8080 --name registry-web --link registry-srv -e REGISTRY_URL=http://registry-srv:5000/v2 -e REGISTRY_NAME=localhost:5000 hyper/docker-registry-web
  ```

## Install Knative and Istio

https://github.com/knative/docs/blob/master/install/Knative-with-any-k8s.md

1. Install Istio

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/istio-crds.yaml && \
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/istio.yaml
```
`kubectl label namespace default istio-injection=enabled`

`kubectl get pods --namespace istio-system`

If you see an error when creating resources about an unknown type, run the `kubectl apply` command again


2. Install Knative and its dependencies

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/serving.yaml \
--filename https://github.com/knative/eventing/releases/download/v0.3.0/release.yaml
```

To check the status of the Knative install components run

```
kubectl get pods --namespace knative-serving
kubectl get pods --namespace knative-eventing
```

## Custom domain setup

IMPORTANT:

`kubectl edit cm config-domain --namespace knative-serving`

  - Use YOUR_IP.nip.io in place of example.com
  - Run `echo $(ifconfig | grep "inet 9." | cut -d ' ' -f2).nip.io` to get the address.

## Install Knative build-pipeline with patch

1. Ensure you have the listed required tools installed https://github.com/knative/build-pipeline/blob/master/DEVELOPMENT.md#requirements

2. Clone the repository and export docker repo for `ko`

```
cd $GOPATH/src/github.com/knative
git clone https://github.com/dibbles/build-pipeline.git
```
  
`export KO_DOCKER_REPO=localhost:5000/knative`

`cd build-pipeline`

3. Install the Knative build-pipeline components

`ko apply -f ./config`

4. Check pods in `knative-build-pipeline` for status of install

## Eventing-sources patch 

1. Clone the repository

```
cd $GOPATH/src/github.com/knative
git clone https://github.com/dibbles/eventing-sources.git
```

`cd eventing-sources`

2. Apply the changes:

`ko apply -f config/default.yaml`

## Fork the sample app 

Fork `github.ibm.com/swiss-cloud/sample` app in GHE to your own org. Keep the name 'sample'. 

## Install knative-pipeline-events:

1. Clone the repository 

```
cd $GOPATH/src/github.ibm.com/swiss-cloud
git clone https://github.ibm.com/swiss-cloud/sound-of-devops.git
```

`cd sound-of-devops`

2. Install the components

`kubectl apply -f ./config`

3. Build the event handler image and push it to your own Dockerhub repository

`docker build -t docker.io/YOUR_DOCKERHUB_ID/github-event-handler .`  

`docker push docker.io/YOUR_DOCKERHUB_ID/github-event-handler`

## Modify yaml files for your own configuration 

- Edit the image location in `github-event-handler.yml` replacing your Dockerhub ID 

`kubectl apply -f event_handler/github-event-handler.yml`

- Modify the GitHub Source template `github_source_templates/git_repo.yml` with your own values (All parts in CAPS)

`kubectl apply -f github_source_templates/git_repo.yml`

## Verify

1. Check that a webhook was successfully created for your `sample` repository

If you have an auth-token you can use this to 
```
curl -i -H "Authorization: token YOUR_GITHUB_TOKEN" https://api.github.ibm.com/repos/GITHUB_OWNER/GITHUB_REPO_NAME/hooks
```

Check if there are hooks present, ensure the IP address hooked matches the IP address for your cluster.  

`ifconfig | grep "inet 9." | cut -d ' ' -f2`


2. Commit a code change to your repository

3. Monitor with `watch kubectl get pods` 

  - You should have pods for the below elements
    1. A Github event handler
    2. A Github event source

    3. Pipeline build will run through its init containers followed by pipeline deploy
    4. Once the pipelines have run through successfully, your running application
 
 
 ## Using the pipeline

 - After performing the `kubectl apply -f github_source_templates/git_repo.yml` you should find a webhook on your project in github.ibm.com (note - currently limited to using github.ibm.com if using this code base and instructions)

 - The application of the above yaml, will also have created a ksvc which is poked by the webhook when code changes are made

 - Delivering a code change to your repository will cause the webhook to poke the ksvc which starts a pod which in turn pokes the ksvc for the event handler (created when you performed the `kubectl apply -f event_handler/github-event-handler.yml`)

 - The pod for the event handler spins up and creates a knative pipelinerun and knative resource(s), these trigger the pipeline code and the the first pipeline pod spins up that is responsible for building your container and pushing to `host.docker.internal:5000/knative/{{.NAME}}:{{.SHORTID}}`, where NAME is you repo name and the SHORTID is the short commit id

 - Once completed, a second pod spins up which handles deploying your application as per your yaml in your apps config directory.

 ## Manual Trigger Of Build

 - There is an endpoint on the event handler service that can be used to trigger a build manually via something like postman - you need to perform a `kubectl get ksvc` and find the DOMAIN (endpoint address) setting of the github-event-pipeline.

 - Using something like postman - POST to <KSVC_DOMAIN>/manual with Header of Content-Type: application/json and a body of:

 ```
 {
   "repourl": YOUR_REPOSITORY_URL,
   "commitid": LONG_COMMIT_ID,
   "reponame": YOUR_REPO_NAME,
   "branch": YOUR_BRANCH_NAME     
}
```

as an example, to build at a specific commit

```
 {
   "repourl": "https://github.ibm.com/APPLEBYD/node-sample",
   "commitid": "518974ca68df99b336be4dafb7edaa3d5e1adeb0",
   "reponame": "node-sample"
   "branch": ""     
}
```

as an example, to build the branch called test

```
 {
   "repourl": "https://github.ibm.com/APPLEBYD/node-sample",
   "commitid": "",
   "reponame": "node-sample"
   "branch": "test"     
}
```

- If you specify a value for both a commitid and branch, the commitid will be used

 ## Current restrictions

 - You must have a file called `deployment.yaml` which contains your app deployment as the code is currently hard coded to replace the image listed in this file with that output in the build.

 - Only run a single build at a time (artifacts are not fully uniquely named and problems could occur).
