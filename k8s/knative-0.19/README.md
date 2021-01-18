Knative is a complex beast,

    We want to use Knative Serving because it provides (among many) a few things we want:
     - scale down to zero services/deployments
     - horizontal pod autoscaling based on number of requests in flight
        (we define how many concurrent requests each replica can handle, it does the rest)
     - transparent support for HTTP and gRPC endpoints

    It does much more, it is important to note that the its goals are quite big and broad:
     - completely automate the lifecycle management of managing services in production,
       it handles the complete lifecycle of releasing a new version (you tag an image, it does the rest)

    It has these main concepts:
    - a service
    - a route (or routes to a service)
    - configuration, the desired state
    - revision, a version/revision of service + route + configuration, they are immutable.

    It combines into a single definition the combination of:
     - a deployment
     - a service
     - an ingress endpoint
     - an hpa configuration

     We create "ksvc" instead of services

     Because knative keeps track of container image digests to track the versions of deployments and configurations etc.. it needs special access to the container registry.
     I needed help to make it fetch my images and Carlos Santana provided https://github.com/csantanapr/knative-private-images
     trying the skip-tag-resolution approach

     This page has a very good description of how Knative Serving works: https://knative.dev/docs/serving/knative-kubernetes-services/

    WORKING NOW:
     - container digests work
     - labels work for cluster-local
     - had to remove traefik for kourier to work


