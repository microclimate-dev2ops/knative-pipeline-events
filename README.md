# sound-of-devops

## Prerequisites

- Docker for Mac - switch to edge version
    - Under advanced set CPU 6, Memory 10, Swap 2.5
    - Under Daemon add insecure registry - host.docker.internal:5000
    - Enable Kubernetes

- Set up a local docker registry 

  ```
  docker run -d -p 5000:5000 --name registry-srv -e REGISTRY_STORAGE_DELETE_ENABLED=true registry:2

  docker run -it -p 8080:8080 --name registry-web --link registry-srv -e REGISTRY_URL=http://registry-srv:5000/v2 -e REGISTRY_NAME=localhost:5000 hyper/docker-registry-web
  ```

## Install Knative and Istio

https://github.com/knative/docs/blob/master/install/Knative-with-any-k8s.md

1. Install Istio

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/istio-crds.yaml && \
kubectl apply --filename https://github.com/knative/serving/releases/
download/v0.3.0/istio.yaml
```
`kubectl label namespace default istio-injection=enabled`

`kubectl get pods --namespace istio-system`

If you see an error when creating resources about an unknown type, run the `kubectl apply` command again


2. Install Knative and its dependencies

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/serving.yaml \
--filename https://github.com/knative/build/releases/download/v0.3.0/release.yaml \
--filename https://github.com/knative/eventing/releases/download/v0.3.0/release.yaml \
--filename https://github.com/knative/eventing-sources/releases/download/v0.3.0/release.yaml
```

To check the status of the Knative install components run

```
kubectl get pods --namespace knative-serving
kubectl get pods --namespace knative-build
kubectl get pods --namespace knative-eventing
kubectl get pods --namespace knative-sources
```

## Custom domain setup

IMPORTANT:

`kubectl edit cm config-domain --namespace knative-serving`

  - Use YOUR_IP.nip.io in place of example.com
  - Run `ifconfig | grep "inet 9."` to get your ip address when using Docker for Mac

## Install Knative build-pipeline

1. Ensure you have the listed required tools installed https://github.com/knative/build-pipeline/blob/master/DEVELOPMENT.md#requirements

2. Clone the repository and export docker repo for ko 

`git clone https://github.com/knative/build-pipeline.git` to GOPATH/github.com/knative
  
`Export KO_DOCKER_REPO=localhost:5000/knative`

`cd build-pipeline`

3. Install the Knative build-pipeline components

`Ko apply -f ./config`

4. Check pods in `knative-build-pipeline` for status of install

## Eventing-sources patch 

1. Clone the repository

`git clone https://github.com/dibbles/eventing-sources.git` into gopath GOPATH/github.com/knative

`cd eventing-sources`

2. Apply the changes:

`ko apply -f config/default.yaml`

## Fork the sample app 

Fork `github.ibm.com/swiss-cloud/sample` app in GHE to your own org. Keep the name 'sample'. 

## Install sound-of-devops:

1. Clone the repository 

`git clone https://github.ibm.com/swiss-cloud/sound-of-devops.git` to GOPATH/github.ibm.com/swiss-cloud

`cd sound-of-devops`

2. Install the components

`Kubectl apply -f ./config`

3. Build the event handler image and push it to your own Dockerhub repository

`docker build -t docker.io/YOUR_DOCKERHUB_ID/github-event-handler .`  

`docker push docker.io/YOUR_DOCKERHUB_ID/github-event-handler`

## Modify yaml files for your own configuration 

- Edit the image location in `github-event-handler.yml` replacing your Dockerhub ID 

`Kubectl apply -f event_handler/github-event-handler.yml`

- Modify the GitHub Source template `github_source_templates/git_repo.yml` with your own values (All parts in CAPS)

`Kubectl apply -f github_source_templates/git_repo.yml`

## Verify

1. Check that a webhook was successfully created for your `sample` repository

2. Commit a code change to your repository

3. Monitor with `watch kubectl get pods` 

  - You should have pods for the below elements
    - Github event handler
    - Github event source

    - Pipeline build 
    - Pipeline deploy
    - Your running application
 