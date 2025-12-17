# Kubernetes Deployment Guide

This guide covers deploying the LLM Verifier application on Kubernetes clusters.

## Prerequisites

- Kubernetes 1.24+
- kubectl configured
- Helm 3.0+ (recommended)
- PV/PVC support for persistent storage
- Ingress controller (nginx, traefik, etc.)

## Quick Start with Helm

1. **Add the Helm repository:**
   ```bash
   helm repo add llm-verifier https://charts.llm-verifier.io
   helm repo update
   ```

2. **Install with default values:**
   ```bash
   helm install llm-verifier llm-verifier/llm-verifier
   ```

3. **Verify deployment:**
   ```bash
   kubectl get pods
   kubectl get svc
   ```

## Manual Kubernetes Deployment

### 1. Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: llm-verifier
  labels:
    name: llm-verifier
```

### 2. ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-verifier-config
  namespace: llm-verifier
data:
  config.yaml: |
    server:
      port: 8080
      host: 0.0.0.0
    database:
      type: sqlite
      path: /data/database.db
    logging:
      level: info
      format: json
    monitoring:
      enabled: true
      prometheus: true
```

### 3. Secrets

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: llm-verifier-secrets
  namespace: llm-verifier
type: Opaque
data:
  # Base64 encoded values
  jwt-secret: <base64-jwt-secret>
  database-encryption-key: <base64-encryption-key>
  openai-api-key: <base64-openai-key>
  anthropic-api-key: <base64-anthropic-key>
```

### 4. Persistent Volume Claim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: llm-verifier-data
  namespace: llm-verifier
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard
```

### 5. Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier
  namespace: llm-verifier
  labels:
    app: llm-verifier
spec:
  replicas: 2
  selector:
    matchLabels:
      app: llm-verifier
  template:
    metadata:
      labels:
        app: llm-verifier
    spec:
      containers:
      - name: llm-verifier
        image: llm-verifier:latest
        ports:
        - containerPort: 8080
          name: http
        env:
        - name: CONFIG_FILE
          value: "/config/config.yaml"
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: llm-verifier-secrets
              key: jwt-secret
        - name: DATABASE_ENCRYPTION_KEY
          valueFrom:
            secretKeyRef:
              name: llm-verifier-secrets
              key: database-encryption-key
        volumeMounts:
        - name: config
          mountPath: /config
        - name: data
          mountPath: /data
        resources:
          requests:
            memory: "512Mi"
            cpu: "500m"
          limits:
            memory: "1Gi"
            cpu: "1000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: llm-verifier-config
      - name: data
        persistentVolumeClaim:
          claimName: llm-verifier-data
```

### 6. Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: llm-verifier-service
  namespace: llm-verifier
spec:
  selector:
    app: llm-verifier
  ports:
  - name: http
    port: 80
    targetPort: 8080
  type: ClusterIP
```

### 7. Ingress

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: llm-verifier-ingress
  namespace: llm-verifier
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: "letsencrypt-prod"
spec:
  ingressClassName: nginx
  tls:
  - hosts:
    - api.llm-verifier.com
    secretName: llm-verifier-tls
  rules:
  - host: api.llm-verifier.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: llm-verifier-service
            port:
              number: 80
```

## High Availability Configuration

### Multi-zone Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier-ha
  namespace: llm-verifier
spec:
  replicas: 3
  selector:
    matchLabels:
      app: llm-verifier
  template:
    metadata:
      labels:
        app: llm-verifier
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
          - labelSelector:
              matchLabels:
                app: llm-verifier
            topologyKey: kubernetes.io/hostname
      topologySpreadConstraints:
      - maxSkew: 1
        topologyKey: topology.kubernetes.io/zone
        whenUnsatisfiable: DoNotSchedule
        labelSelector:
          matchLabels:
            app: llm-verifier
```

### Database with StatefulSet

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: llm-verifier-db
  namespace: llm-verifier
spec:
  serviceName: llm-verifier-db
  replicas: 2
  selector:
    matchLabels:
      app: llm-verifier-db
  template:
    metadata:
      labels:
        app: llm-verifier-db
    spec:
      containers:
      - name: postgres
        image: postgres:15
        env:
        - name: POSTGRES_DB
          value: llm_verifier
        - name: POSTGRES_USER
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: username
        - name: POSTGRES_PASSWORD
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: password
        volumeMounts:
        - name: data
          mountPath: /var/lib/postgresql/data
  volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes: ["ReadWriteOnce"]
      resources:
        requests:
          storage: 50Gi
```

## Monitoring and Observability

### Prometheus ServiceMonitor

```yaml
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: llm-verifier-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: llm-verifier
  endpoints:
  - port: metrics
    path: /metrics
    interval: 30s
```

### Grafana Dashboard

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: llm-verifier-dashboard
  namespace: monitoring
  labels:
    grafana_dashboard: "1"
data:
  llm-verifier.json: |
    {
      "dashboard": {
        "title": "LLM Verifier",
        "tags": ["llm", "verification"],
        "timezone": "browser",
        "panels": [
          {
            "title": "API Response Time",
            "type": "graph",
            "targets": [
              {
                "expr": "histogram_quantile(0.95, rate(http_request_duration_seconds_bucket[5m]))",
                "legendFormat": "95th percentile"
              }
            ]
          }
        ]
      }
    }
```

## Security

### Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: llm-verifier-netpol
  namespace: llm-verifier
spec:
  podSelector:
    matchLabels:
      app: llm-verifier
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: ingress-nginx
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: llm-verifier-db
    ports:
    - protocol: TCP
      port: 5432
  - to: []
    ports:
    - protocol: TCP
      port: 443  # HTTPS for external API calls
```

### Pod Security Standards

```yaml
apiVersion: policy/v1beta1
kind: PodSecurityPolicy
metadata:
  name: llm-verifier-psp
spec:
  privileged: false
  allowPrivilegeEscalation: false
  requiredDropCapabilities:
    - ALL
  runAsUser:
    rule: MustRunAsNonRoot
  seLinux:
    rule: RunAsAny
  supplementalGroups:
    rule: MustRunAs
    ranges:
    - min: 1
      max: 65535
  fsGroup:
    rule: MustRunAs
    ranges:
    - min: 1
      max: 65535
  readOnlyRootFilesystem: true
  volumes:
  - configMap
  - emptyDir
  - persistentVolumeClaim
  - secret
```

## Backup and Disaster Recovery

### Database Backup CronJob

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: llm-verifier-backup
  namespace: llm-verifier
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: postgres:15
            command:
            - /bin/bash
            - -c
            - |
              pg_dump -h llm-verifier-db -U postgres llm_verifier > /backup/backup-$(date +%Y%m%d-%H%M%S).sql
            env:
            - name: PGPASSWORD
              valueFrom:
                secretKeyRef:
                  name: db-secret
                  key: password
            volumeMounts:
            - name: backup
              mountPath: /backup
          volumes:
          - name: backup
            persistentVolumeClaim:
              claimName: backup-pvc
          restartPolicy: OnFailure
```

### Disaster Recovery

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: disaster-recovery
  namespace: llm-verifier
data:
  restore.sh: |
    #!/bin/bash
    # Restore from latest backup
    LATEST_BACKUP=$(ls -t /backup/backup-*.sql | head -1)
    psql -h llm-verifier-db -U postgres -d llm_verifier < $LATEST_BACKUP
```

## Scaling

### Horizontal Pod Autoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: llm-verifier-hpa
  namespace: llm-verifier
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: llm-verifier
  minReplicas: 2
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

### Cluster Autoscaler

```yaml
apiVersion: cluster-autoscaler.sh/v1beta1
kind: ClusterAutoscaler
metadata:
  name: cluster-autoscaler
  namespace: kube-system
spec:
  scaleDownDelayAfterAdd: 10m
  scaleDownDelayAfterDelete: 10s
  scaleDownDelayAfterFailure: 3m
  scaleDownUnneededTime: 10m
  scaleDownUnreadyTime: 20m
  unremovableNodeRecheckTimeout: 5m
```

## CI/CD Integration

### GitOps with ArgoCD

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: llm-verifier
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/your-org/llm-verifier
    path: k8s/
    targetRevision: HEAD
  destination:
    server: https://kubernetes.default.svc
    namespace: llm-verifier
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

## Troubleshooting

### Common Issues

1. **Pod crashes:**
   ```bash
   kubectl logs -f pod/llm-verifier-xxx -n llm-verifier
   kubectl describe pod llm-verifier-xxx -n llm-verifier
   ```

2. **Database connection issues:**
   ```bash
   kubectl exec -it pod/db-pod -n llm-verifier -- psql -U postgres -d llm_verifier
   ```

3. **Resource constraints:**
   ```bash
   kubectl top pods -n llm-verifier
   kubectl top nodes
   ```

### Debugging Commands

```bash
# Check pod status
kubectl get pods -n llm-verifier

# Check service endpoints
kubectl get endpoints -n llm-verifier

# Check ingress
kubectl describe ingress llm-verifier-ingress -n llm-verifier

# Port forward for local testing
kubectl port-forward svc/llm-verifier-service 8080:80 -n llm-verifier

# Check resource usage
kubectl top pods --containers -n llm-verifier
```

## Performance Optimization

### Resource Optimization

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: llm-verifier-optimized
spec:
  template:
    spec:
      containers:
      - name: llm-verifier
        resources:
          requests:
            memory: "256Mi"
            cpu: "100m"
          limits:
            memory: "512Mi"
            cpu: "500m"
        env:
        - name: GOMAXPROCS
          value: "2"
        - name: GOGC
          value: "50"
```

### Network Optimization

```yaml
apiVersion: v1
kind: Service
metadata:
  name: llm-verifier-optimized
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: 'true'
spec:
  type: LoadBalancer
  externalTrafficPolicy: Local  # Preserve client IP
```