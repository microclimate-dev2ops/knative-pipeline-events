# sound-of-devops

Instructions For Using:

Docker for Mac
  edge
  Insecure registry host.docker.internal:5000
  Kube enable
  Cpu 6
  Mem 10
  Swap 2.5

docker run -d -p 5000:5000 --name registry-srv -e REGISTRY_STORAGE_DELETE_ENABLED=true registry:2
docker run -it -p 8080:8080 --name registry-web --link registry-srv -e REGISTRY_URL=http://registry-srv:5000/v2 -e REGISTRY_NAME=localhost:5000 hyper/docker-registry-web 
Delete existing containers re run


Install native and istio: https://github.com/knative/docs/blob/master/install/Knative-with-any-k8s.md

kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/istio-crds.yaml && \
kubectl apply --filename https://github.com/knative/serving/releases/
download/v0.3.0/istio.yaml

If you see an error when creating resources about an unknown type, run the second kubectl apply command again

kubectl label namespace default istio-injection=enabled

kubectl get pods --namespace istio-system

kubectl apply --filename https://github.com/knative/serving/releases/download/v0.3.0/serving.yaml \
--filename https://github.com/knative/build/releases/download/v0.3.0/release.yaml \
--filename https://github.com/knative/eventing/releases/download/v0.3.0/release.yaml \
--filename https://github.com/knative/eventing-sources/releases/download/v0.3.0/release.yaml

kubectl get pods --namespace knative-serving
kubectl get pods --namespace knative-build
kubectl get pods --namespace knative-eventing
kubectl get pods --namespace knative-sources
kubectl get pods --namespace knative-monitoring



Install build-pipeline:  https://github.com/knative/build-pipeline/blob/master/DEVELOPMENT.md

  Clone build-pipeline ... go path etc
  Export KO_DOCKER_REPO=localhost:5000
  Ko apply -f ./config


Clone github.com/dibbles/eventing-sources - into gopath ... src/github.com/knative
Cd eventing-sources
ko apply -f config/default.yaml


Fork github.ibm.com/APPLEBYD/simple app in GHE to your own org


Install sound-of-devops:

  Clone
  Kubectl apply -f ./config
  docker build -t docker.io/dibbles/github-event-handler .    (PROBABLY DONT NEED TO TAG TO DOCKER.IO)
  Docker push above
  Modify event_handler yaml image
  Kubectl apply -f (the above)
  Modify GitHub source template
  Kubectl apply -f the above

