# What is Knative (Serving)

Well, it is EVERYTHING
"""
Kubernetes-based platform to deploy and manage modern serverless workloads.
"""
https://knative.dev/

## ELI5 ?

I'm going to only cover Knative Serving.
https://knative.dev/docs/serving/

Basically, it is a higher level abstraction to manage and operate service deployments on k8s.
It is focussed on automating away the operations side as much as possible and handles whole JTBDs such as triggering deployment rollouts w/ blue/green or whatnot just by CI/CD tagging the container.. (or close..)

The Knative Serving project provides middleware primitives that enable:
1. Rapid deployment of serverless containers
1. Automatic scaling up and down to zero
   And HPA scaling based on number of concurrent requests in flight per replica/pod (yeah!)
1. Routing and network programming for Istio components
1. Point-in-time snapshots of deployed code and configurations


## Scale down to zero:

Knatives makes this MAGICAL, so you have a service and if there are no requests, it isn't up.
It supports from the get-go "serverless" services (config is: https://knative.dev/docs/serving/autoscaling/scale-to-zero/)
- Native & transparent support for HTTP and gRPC endpoints. (but doesn't support queue based services, for that they have "Knative Eventing" .. which is a whole different offering.)

So, the way it works is this:
 https://github.com/knative/serving/blob/master/docs/spec/overview.md

Important concepts:
- A new "service" abstraction
- a route (or routes to a service)
- configuration, the desired state
- revision, a version/revision of service + route + configuration, they are immutable.

It combines into a single definition the combination of:
 - a deployment
 - a service
 - an ingress endpoint
 - an hpa configuration
[Concepts](https://twitter.com/bibryam/status/1254355796172865536/photo/1)

We create "ksvc" instead of services

## Architecture implications

Well, we need we don't need to change to an event driven architecture. We continue to develop services. But we deploy "knative services", so our CI/CD needs to change, our k8s definitions would change too (but will be simpler, shorter!)

We gain this by leveraging a variety of new critical systems and are now on the hot-path of our requests.
We also must use new tools to manually debug or explore our k8s deployments (as you'll see).
The new abstration layers are like always, leaky, if things don't work as expected, someone must peel the layers and understand/debug the problems.

Some "basic" images of how Knative works..
[Diagram](https://blog.nebrass.fr/wp-content/uploads/knative-serving-ecosystem-1199x1536.png)
[Network Ingress](https://developers.redhat.com/blog/2020/06/30/kourier-a-lightweight-knative-serving-ingress/)


## Organizational Implications

We must upgrade our k8s clusters to at least k8s 1.17 (we're at 1.15)
We have very few SREs, our developer/researcher to SRE/systems-engineer ratio is extremely high.
We must now maintain a new critical system that needs the following moving parts, properly configured and always rock-solid

The instalation isn't trivial:
https://knative.dev/docs/install/any-kubernetes-cluster/
It includes:
1. k8s CRDs
1. Knative Service Core components
1. A networking layer, from which there are 6 to choose from (which should we use? what are our criterias?) (Kourier is what I'm using [Kourier](https://developers.redhat.com/blog/2020/06/30/kourier-a-lightweight-knative-serving-ingress/)
1. DNS integration (Knative Services integrated all the way up to DNS and Ingress endpoints)
1. HPA integration
1. TLS integration

Effectively, we're pushing a lot of complexity to lower layers of the stack and this has the consequence of pushing the problem to SRE-land.

PS: I couldn't get this to work, @Bruno was the one that fixed my setup! (again, SRE-land expertize required...)

Compare this to KEDA, where we (sw-devs) own the management and maintenance of queues, 200-LoC sidecars, and a few extra HPA definitions.

## Applying this to Maestro

Well, we would create new definitions of our deployments using knative 'ksvc's
We would also change our URL endpoints for our services and update Model-Management accordingly
We would need have multi-k8s deployments and have http/gRPC traffic between them

## DEMO TIME

...

## PROBLEMZZ


1. Is not simple (which is a core engineering principle I push for)
1. the burden of operating, maintaining and debuging issues, 98% would land on the SRE team..
1. Spending 2 "innovation tokens" (network service/mesh/thingie + knative Serving)

Need to discuss better w/ SRE team (and VP-Eng) appetite for adopting & supporting this.
We still need to identify the "scale from zero" lag with more testing. Our SLO is <10s, 15s might be okay

### Miguel's problems on setting it up

- My local k8s already had an ingress controller that collided w/ the networking layer that knative needed.
  - Bruno spotted this problem (that I was not understanding at all) and we fixed easily
  - had to remove traefik for kourier to work
- I'm using plain http container registries (like `sudo microk8s enable registry`)
  Because knative keeps track of container image digests to track the versions of deployments and configurations etc.. it needs special access to the container registry.
  I needed help to make it fetch my images and Carlos Santana provided https://github.com/csantanapr/knative-private-images
  - fixed by using container digests on the image definition
- By default knative creates public ingress endpoints for services, this a problem:
   - security, we want things cluster local by default, always.
   - svc2svc communication was failing
   - the documentation on how to make them cluster local isn't super clear
     (https://knative.dev/docs/serving/cluster-local-route/)
     no yaml example of service definition that is always cluster local, only adding a "label" to make it so. The bug in the definition I had was "label", but was "labels"
