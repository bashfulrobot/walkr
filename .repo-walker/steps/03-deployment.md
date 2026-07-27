---
title: "Deploying it: the *k8s manifest*"
label: Deployment
kind: Config
order: 3
layout: config
summary: Config gets the same treatment as code — annotated inline, not left to speak for itself.
---
```yaml mark=4,9,11
apiVersion: apps/v1
kind: Deployment
spec:
  replicas: 2
  template:
    spec:
      containers:
        - name: repo-walker
          readinessProbe:
            httpGet: { path: /healthz, port: 8080 }
          resources:
            limits: { cpu: "200m", memory: "128Mi" }
```
1. Two replicas — this is a static-site server, so redundancy is cheap and mostly guards against node drain.
2. Without this, a slow-starting pod can receive traffic before the site is built and served.
3. Deliberately tight — the binary embeds all assets, so there's no separate asset-serving footprint to budget for.
