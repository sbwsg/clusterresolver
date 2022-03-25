# Cluster-Scoped Task and Pipeline Resolver

This Resolver implements ClusterTask and ClusterPipeline support for
Tekton Pipelines.

## Resolver Type

Use Resolver type `clusterscoped`

## Parameters

| Param Name | Description                                                       |
|------------|-------------------------------------------------------------------|
| `kind`     | Either `task` or `pipeline`                                       |
| `name`     | The name of the cluster-scoped Task or Pipeline. e.g. `git-clone` |

## How it works:
- a single namespace is created and populated with Tasks and Pipelines.
  This is where "cluster-scoped" Tasks and Pipelines should be put by
  the operator/admin.
- this Resolver is given exclusive access to read those Tasks and
  Pipelines from that private namespace.
- when a taskref or pipelineref in any namespace references the
  `clusterscoped` resolver, it will look up that task or pipeline from its
  private namespace.
- if the task or pipeline exists in the private namespace then it's
  returned to Tekton Pipelines. If not, the resolution request fails.

## Getting Started

### Requirements

- A cluster running this [in-progress pull request of Tekton Pipelines](https://github.com/tektoncd/pipeline/pull/4596)
  with the `alpha` feature gate enabled.
- `ko` installed.
- The `tekton-remote-resolution` namespace and `ResolutionRequest`
  controller installed. See [../README.md](../README.md).

### Install

1. Install the clusterscoped Resolver:

```bash
$ ko apply -f ./config
```

### Setup and Trying it Out

First, install the `git-clone` Task from the Catalog into the private
namespace:

```bash
# Install the git-clone task into the clusterresolver's private namespace
$ kubectl apply -n tekton-cluster-scoped-resources -f https://raw.githubusercontent.com/tektoncd/catalog/main/task/git-clone/0.5/git-clone.yaml
```

Next, create some ResolutionRequests for cluster-scoped tasks:

```bash
# This creates two ResolutionRequests, one which will succeed and one
# which will fail.
$ kubectl apply -n default -f ./test-clusterresolver.yaml
```

Now when you look at the ResolutionRequests you'll see that the request
for the git-clone Task succeeded, even though that Task lives in the
clusterscoped resolver's private namespace:

```bash
$ kubectl get resolutionrequest -n default -w
NAME                               SUCCEEDED   REASON
get-git-clone-cluster-task         True
try-getting-unknown-cluster-task   False       ResolutionFailed
```

The git-clone Task is now "cluster-scoped"; effectively working as a
`ClusterTask`.

### Example PipelineRun

First, install this repo's "simple pipeline" into the clusterscoped resolver's private namespace:

```bash
$ kubectl apply -n tekton-cluster-scoped-resources -f simple-pipeline.yaml
```

Now you'll need to create a PipelineRun that uses the `clusterscoped`
resolver:

```yaml
apiVersion: tekton.dev/v1beta1
kind: PipelineRun
metadata:
  name: cluster-scoped-demo
spec:
  pipelineRef:
    resolver: clusterscoped
    resource:
    - name: kind
      value: pipeline
    - name: name
      value: simple-pipeline
  params:
  - name: name
    value: cluster-scoped
```

## What's Supported?

- Fetching `Tasks` and `Pipelines` cluster-wide.

---

Except as otherwise noted, the content of this page is licensed under the
[Creative Commons Attribution 4.0 License](https://creativecommons.org/licenses/by/4.0/),
and code samples are licensed under the
[Apache 2.0 License](https://www.apache.org/licenses/LICENSE-2.0).
