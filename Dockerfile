#*******************************************************************************
# Licensed Materials - Property of IBM
# "Restricted Materials of IBM"
# 
# Copyright IBM Corp. 2018 All Rights Reserved
#
# US Government Users Restricted Rights - Use, duplication or disclosure
# restricted by GSA ADP Schedule Contract with IBM Corp.
#*******************************************************************************
FROM golang:1.9 as builder
USER root
RUN curl -fsSL -o /usr/local/bin/dep https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 && chmod +x /usr/local/bin/dep
WORKDIR /go/src/github.ibm.com/swiss-cloud/sound-of-devops
COPY . .
RUN dep ensure -vendor-only -v
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o knative-devops-runtime .

FROM alpine@sha256:7df6db5aa61ae9480f52f0b3a06a140ab98d427f86d8d5de0bedab9b8df6b1c0
RUN apk --update add curl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl
RUN chmod +x ./kubectl
RUN mv ./kubectl /usr/local/bin/kubectl
CMD mkdir /knative-devops
WORKDIR /knative-devops
COPY --from=builder /go/src/github.ibm.com/swiss-cloud/sound-of-devops/knative-devops-runtime /knative-devops
COPY --from=builder /go/src/github.ibm.com/swiss-cloud/sound-of-devops/templates /knative-devops/templates
EXPOSE 8080
CMD /knative-devops/knative-devops-runtime