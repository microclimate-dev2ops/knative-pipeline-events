# sound-of-devops

## Prerequisites

- Docker for Mac - switch to edge version
    - Under advanced set CPU 6, Memory 10, Swap 2.5
- Under Daemon add insecure registry - host.docker.internal:5000
- Enable Kubernetes

Set up a local docker registry 

```
docker run -d -p 5000:5000 --name registry-srv -e REGISTRY_STORAGE_DELETE_ENABLED=true registry:2

docker run -it -p 8080:8080 --name registry-web --link registry-srv -e REGISTRY_URL=http://registry-srv:5000/v2 -e REGISTRY_NAME=localhost:5000 hyper/docker-registry-web
```

## Install Knative and Istio

https://github.com/knative/docs/blob/master/install/Knative-with-any-k8s.md

Install Istio

```
kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/istio-crds.yaml && \
kubectl apply --filename https://github.com/knative/serving/releases/
download/v0.3.0/istio.yaml
```
`kubectl label namespace default istio-injection=enabled`

`kubectl get pods --namespace istio-system`

If you see an error when creating resources about an unknown type, run the `kubectl apply` command again


Install Knative and its dependencies with the below `kubectl apply`

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

## Install Knative build-pipeline

https://github.com/knative/build-pipeline/blob/master/DEVELOPMENT.md

`git clone https://github.com/knative/build-pipeline.git` to GOPATH/github.ibm.com/swiss-cloud
  
`Export KO_DOCKER_REPO=localhost:5000/knative`

Install the Knative components 

`Ko apply -f ./config`

## Eventing-sources patch 

`git clone github.com/dibbles/eventing-sources` into gopath GOPATH/github.com/knative

`cd eventing-sources`

`ko apply -f config/default.yaml`

## Fork the sample app 

Fork `github.ibm.com/swiss-cloud/sample` app in GHE to your own org. Keep the name 'sample'. 

## Install sound-of-devops:

`git clone https://github.ibm.com/swiss-cloud/sound-of-devops.git` to GOPATH/github.ibm.com/swiss-cloud

`cd sound-of-devops`

Install the components

`Kubectl apply -f ./config`

Build the event handler image, pushing to your own dockerhub repository

`docker build -t docker.io/YOUR_DOCKERHUB_ID/github-event-handler .`  

`docker push docker.io/YOUR_DOCKERHUB_ID/github-event-handler`

## Modify yaml files for your own configuration 

Edit the image location in `github-event-handler.yml` replacing your dockerhub ID 

`Kubectl apply -f event_handler/github-event-handler.yml`

Modify the GitHub Source template `github_source_templates/git_repo.yml` with your own values (All parts in CAPS)

`Kubectl apply -f github_source_templates/git_repo.yml`


`watch kubectl get pods` 