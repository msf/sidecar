# What is KEDA

Kubernetes Event Driven Autoscaler
https://keda.sh

## ELI5 ?

Well, KEDA is a system to trigger the spin up (or spin down) of deployments based on "queue sizes".
It assumes one emits events and you scale your deployment based on that. You can use the events as work-items too (which implies an event driven architecture)

[Diagram](https://cottonglow.github.io/2019-12-02-exploring-keda/)

## Scale down to zero:

Keda makes this super easy, we just define a queue we want to use as "trigger" to a deployment
based on that queue size, it will keep the deployment at zero if the queue is empty.
It will activate the deployment and also scale it based on how many messages are pending in the queue
We'll use event driven architecture, some systems emit events, other systems consume.. this is how the communicate.

[Architecture](https://keda.sh/docs/2.0/concepts/#architecture)

It supports many types of queue/stream systems:
- Kafka
- AWS SQS
- RabbitMQ
- Redis lists or streams..etc..
- *but also events that are just trigggers, like cloudwatch, logs, etc...


## Architecture implications

Well, we need to use an event driven architecture and uses queues to communicate between services..
- Which isn't always natural or the right thing to do.
- Even if it is applicable and a good fit, we need to change our code... .. or do we ?
- We'll need to upgrade our k8s deployments to v1.16 (we're at v1.15)

## Applying this to Maestro

Maestro has moved the MT & QE systems to reqs/response synchronous architecture, it was "message oriented" before..
I was the one doing a big push for this, because:
1. It is simpler to develop req/response synchronous systems.
1. It greatly improves visibility into error rates and stability problems
1. MT & QE are pure machine-based system, a req/response is just fine and is a simpler approach to design, without complications of queues and callback flows..etc..
1. Reduces the number of required systems and SPOFs (no need for queueing system)

Now we're talking about GOING BACK TO QUEUE-BASED archicture!! eheheheh
But hey, it is for a very valid product-based need!
We still want to keep some parts of the old architecture:
- keep the req/response abstraction: easier to develop, higher visibility into error rates.
- minimize the addition of complexity or moving parts..

Additionally:
- we would add some new definitions of our deployments using keda autoscaling definitions
- We wouldn't necessarily need to change our service endpoints nor Model-Management.
- We would need have multi-k8s deployments and have rabbitMQ traffic between them


## Maestro w/ SIDECARS!

The core idea is:
1. lets keep maestro w/ the same web (or gRPC) server endpoints.
1. maestro server side, add a "sidecar" process that consumes from a queue, talks to maestro, returns the response to the sender (using also a queue).
1. client side, option 1: have library code that is a req/resp library but uses queues under the hood
1. on the client side, option 2: a sidecar, pretending to be maestro, that receives a HTTP request and sends it to a queue, then waits for the response

## DEMO TIME


## PROBLEMZZ

Well, we need to maintain sidecar code and ensure it is rock-solid.
We also need to make our sidecars support gRPC (which it doesn't right now)
