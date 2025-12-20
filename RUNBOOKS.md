# Enhanced LLM Verifier Production Runbooks

## 📋 Table of Contents

1. [Quick Start](#quick-start)
2. [Deployment Operations](#deployment-operations)
3. [Monitoring Procedures](#monitoring-procedures)
4. [Emergency Response](#emergency-response)
5. [Maintenance Tasks](#maintenance-tasks)
6. [Scaling Operations](#scaling-operations)
7. [Backup and Recovery](#backup-recovery)
8. [Security Procedures](#security-procedures)
9. [Troubleshooting Playbooks](#troubleshooting-playbooks)

## 🚀 Quick Start

### Emergency Deployment (5-minute rollback)

```bash
# 1. Quick rollback to previous version
kubectl rollout undo deployment/llm-verifier

# 2. Scale down immediately
kubectl scale deployment llm-verifier --replicas=1

# 3. Health check
kubectl wait --for=condition=available --timeout=60s deployment/llm-verifier

# 4. Verify endpoints
curl -f http://api.llm-verifier.com/health

# 5. Check logs for issues
kubectl logs --tail=50 deployment/llm-verifier --since=5m
```

### Standard Deployment (15-minute deployment)

```bash
# 1. Update to new version
kubectl set image deployment/llm-verifier llm-verifier:v1.0.1

# 2. Apply updated configuration
kubectl apply -f k8s/updated-deployment.yaml

# 3. Wait for rollout completion
kubectl rollout status deployment/llm-verifier --timeout=300s

# 4. Scale up to target replicas
kubectl scale deployment llm-verifier --replicas=3

# 5. Wait for pods to be ready
kubectl wait --for=condition=ready --timeout=300s pod -l app=llm-verifier

# 6. Verify deployment
kubectl get pods -l app=llm-verifier

# 7. Test endpoints
curl -f http://api.llm-verifier.com/health
curl -f http://api.llm-verifier.com/metrics
```

## 📋 Deployment Operations

### Blue-Green Deployment

```bash
# 1. Deploy green version (canary)
kubectl apply -f k8s/canary-deployment.yaml

# 2. Monitor green deployment
kubectl get pods -l version=green
kubectl logs -l version=green deployment/canary-llm-verifier --tail=10

# 3. Test green endpoints
GREEN_POD=$(kubectl get pods -l version=green -o jsonpath='{.items[0].metadata.name}')
kubectl port-forward $GREEN_POD 8080:8081 &
GREEN_PID=$!

# Test green endpoint
curl -f http://localhost:8081/health

# 4. Promote green to production (if healthy)
if curl -s http://localhost:8081/health | grep -q "healthy"; then
    # Switch traffic to green
    kubectl patch service llm-verifier-service -p '{"spec":{"selector":{"app":{"version":"green"}}}'
    echo "✅ Green deployment promoted"
else
    # Rollback green deployment
    kubectl delete deployment canary-llm-verifier
    echo "⚠️  Green deployment rolled back"
fi

kill $GREEN_PID
```

### Rolling Update Strategy

```bash
# 1. Start rolling update
kubectl set image deployment/llm-verifier llm-verifier:v1.0.1

# 2. Monitor rollout progress
kubectl rollout status deployment/llm-verifier --timeout=600s

# 3. Check replica sets
kubectl get replicasets
kubectl get pods -l app=llm-verifier

# 4. Verify update completion
kubectl rollout status deployment/llm-verifier
kubectl get pods -l app=llm-verifier -o wide
```

## 📊 Monitoring Procedures

### Health Monitoring

```bash
# 1. Check overall cluster health
kubectl get pods --all-namespaces
kubectl get events --all-namespaces --sort-by='.lastTimestamp'

# 2. Application-specific health
curl -f http://api.llm-verifier.com/health
curl -f http://api.llm-verifier.com/ready
curl -f http://api.llm-verifier.com/metrics

# 3. Database health
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "SELECT 1;"
kubectl exec -it redis-xxx -- redis-cli ping

# 4. Infrastructure monitoring
kubectl top nodes
kubectl top pods -l app=llm-verifier
kubectl get hpa
kubectl get pvc
```

### Performance Monitoring

```bash
# 1. Real-time performance metrics
kubectl exec -it deployment/llm-verifier-xxx -- curl -s http://localhost:8080/metrics

# 2. Database performance
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "
SELECT 
    schemaname,
    tablename,
    indexname,
    idx_scan,
    idx_tup_fetch,
    idx_tup_read,
    idx_tup_deleted
FROM pg_stat_user_tables 
WHERE schemaname = 'public'
ORDER BY schemaname, tablename;
"

# 3. Resource usage analysis
kubectl describe pods -l app=llm-verifier
kubectl get nodes -o wide

# 4. Network performance
kubectl get networkpolicies
kubectl get ingress
```

### Log Analysis

```bash
# 1. Real-time log streaming
kubectl logs -f deployment/llm-verifier --tail=100

# 2. Error analysis
kubectl logs deployment/llm-verifier --since=1h | grep ERROR | wc -l
kubectl logs deployment/llm-verifier --since=1h | grep WARN | wc -l

# 3. Security events
kubectl get events --field-selector=reason=FailedCreate --sort-by='.lastTimestamp'
kubectl get events --field-selector=type=Warning --sort-by='.lastTimestamp'

# 4. Performance bottlenecks
kubectl logs deployment/llm-verifier | grep -i "slow\|timeout\|error" | tail -10
```

## 🚨 Emergency Response

### Critical Incident Response

```bash
# 1. Immediate containment
kubectl scale deployment llm-verifier --replicas=0
kubectl annotate deployment/llm-verifier incident="security-breach-$(date +%s)"

# 2. Enable debug mode
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","env":[{"name":"DEBUG","value":"true"}]}}}}'

# 3. Security audit
kubectl auth can-i --list --as=system:serviceaccount
kubectl get events --field-selector=type=Warning --since=10m

# 4. Notification
curl -X POST -H "Content-Type: application/json" \
  -d '{"level":"critical","message":"Security incident detected","timestamp":"'$(date -Iseconds)'"}' \
  https://alerts.llm-verifier.com/webhook

# 5. Documentation
echo "INCIDENT RESPONSE CHECKLIST:" > incident-$(date +%Y%m%d).log
echo "✅ Deployment scaled down" >> incident-$(date +%Y%m%d).log
echo "✅ Debug mode enabled" >> incident-$(date +%Y%m%d).log
echo "✅ Security audit completed" >> incident-$(date +%Y%m%d).log
echo "⏰ Incident response completed at $(date)" >> incident-$(date +%Y%m%d).log
```

### Service Degradation Response

```bash
# 1. Identify failing service
kubectl get pods -l app=llm-verifier | grep -E "(CrashLoopBackOff|Error|ImagePullBackOff)"

# 2. Scale up healthy services
kubectl scale deployment llm-verifier --replicas=2

# 3. Enable circuit breaker for failing endpoints
kubectl annotate service llm-verifier-service circuit-breaker="enabled"

# 4. Monitor recovery
watch kubectl get pods -l app=llm-verifier -o wide

# 5. Gradual traffic restoration
kubectl patch service llm-verifier-service -p '{"spec":{"selector":{"app":{"version":"healthy"}}}'
```

## 🔧 Maintenance Tasks

### Routine Maintenance

```bash
# 1. Daily health check
curl -f http://api.llm-verifier.com/health || echo "Health check failed" | mail -s "Daily Health Check" admin@llm-verifier.com

# 2. Log rotation
kubectl logs deployment/llm-verifier --since=24h > /var/log/llm-verifier/health-$(date +%Y%m%d).log

# 3. Database cleanup
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "VACUUM ANALYZE public.verification_results;"

# 4. Resource optimization
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","resources":{"requests":{"memory":"512Mi","cpu":"200m"}}]}}}}'

# 5. Certificate renewal check
kubectl get certificates -n llm-verifier | grep -E "(expiring|expired)" || echo "Certificates OK"
```

### Weekly Maintenance

```bash
# 1. Complete system backup
kubectl exec -it deployment/llm-verifier-xxx -- /app/backup.sh

# 2. Performance analysis
kubectl top pods -l app=llm-verifier --no-headers | awk 'NR>1 {sum+=$6} END {print "Avg CPU:", $6/NR}' > /var/log/weekly-performance-$(date +%Y%m%d).log

# 3. Security scan
trivy image --format json llm-verifier:v1.0.0 > /var/log/security-scan-$(date +%Y%m%d).json

# 4. Resource cleanup
kubectl delete pods -l app=llm-verifier --field-selector=status.phase=Succeeded --since=72h
kubectl delete configmaps -l app=llm-verifier --field-selector=age>7d
```

### Monthly Maintenance

```bash
# 1. Major version upgrade
kubectl set image deployment/llm-verifier llm-verifier:v1.0.2
kubectl rollout status deployment/llm-verifier --timeout=600s

# 2. Security updates
helm upgrade prometheus prometheus-community/kube-prometheus-stack --namespace monitoring
helm upgrade grafana grafana/grafana --namespace monitoring

# 3. Storage optimization
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "REINDEX INDEX public.verification_results;"

# 4. Capacity planning
kubectl get nodes --no-headers | awk '/CPU Capacity/ {cpu+=$2} END {print "Total CPU:", $2}'
kubectl get nodes --no-headers | awk '/Memory Capacity/ {mem+=$2} END {print "Total Memory:", $2}'
```

## 🛠️ Scaling Operations

### Horizontal Scaling

```bash
# 1. Scale up for high load
kubectl patch hpa llm-verifier-hpa -p '{"spec":{"maxReplicas":10,"minReplicas":2}}'

# 2. Monitor scaling events
kubectl get events --field-selector=reason=SuccessfulCreate --sort-by='.lastTimestamp'

# 3. Load testing
kubectl get hpa llm-verifier-hpa -o yaml
kubectl describe hpa llm-verifier-hpa

# 4. Performance tuning
kubectl top pods -l app=llm-verifier
kubectl get nodes
```

### Vertical Scaling

```bash
# 1. Increase resource limits
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","resources":{"limits":{"memory":"4Gi","cpu":"2000m"},"requests":{"memory":"2Gi","cpu":"1000m"}}]}}}}'

# 2. Performance testing
kubectl exec -it deployment/llm-verifier-xxx -- stress-ng --cpu 1 --timeout 30s

# 3. Profile application
kubectl exec -it deployment/llm-verifier-xxx -- curl -s http://localhost:8080/pprof/profile > /tmp/profile.pprof

# 4. Optimize configuration
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","env":[{"name":"GOMAXPROCS","value":"2"},{"name":"GOGC","value":"50"}]}}}}}}'
```

## 🔒 Security Procedures

### Incident Response

```bash
# 1. Security incident response
./security-incident-response.sh --type security-breach --severity critical

# 2. Vulnerability scanning
trivy image --severity HIGH,CRITICAL llm-verifier:v1.0.0
trivy fs --security-checks vuln /app/data

# 3. Audit logging
kubectl get events --field-selector=type=Warning --since=24h
kubectl logs deployment/llm-verifier --since=24h | grep -i "error\|fail\|denied"

# 4. Forensic analysis
kubectl cp deployment/llm-verifier-xxx:/app/logs /tmp/forensic-$(date +%Y%m%d)/
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "SELECT * FROM audit_log WHERE timestamp > NOW() - INTERVAL '24 hours';"
```

### Compliance Auditing

```bash
# 1. SOC 2 Type II audit
./compliance-audit.sh --framework soc2 --output-format json > /tmp/soc2-audit-$(date +%Y%m%d).json

# 2. GDPR compliance check
./gdpr-compliance-check.sh --data-export-format anonymized --consent-management enabled

# 3. Access control review
kubectl auth can-i --list --as=system:serviceaccount
kubectl get roles -n llm-verifier
kubectl get rolebindings -n llm-verifier

# 4. Encryption verification
kubectl get secrets -n llm-verifier -o yaml
openssl x509 -in /etc/ssl/tls.crt -text -noout | grep -i "subject\|issuer"
```

## 🎯 Troubleshooting Playbooks

### Application Not Starting

```bash
# 1. Check pod status
kubectl get pods -l app=llm-verifier -o wide
kubectl describe pod -l app=llm-verifier

# 2. Check events
kubectl get events -n llm-verifier --sort-by='.lastTimestamp' --tail=20

# 3. Check logs
kubectl logs deployment/llm-verifier --tail=100
kubectl logs -l app=llm-verifier -c llm-verifier-xxx

# 4. Check resource usage
kubectl top pod -l app=llm-verifier
kubectl describe pod -l app=llm-verifier

# 5. Verify configuration
kubectl get configmap llm-verifier-config -o yaml
kubectl get secret llm-verifier-secrets -o yaml

# 6. Network connectivity
kubectl exec -it deployment/llm-verifier-xxx -- nslookup api.llm-verifier.com
kubectl exec -it deployment/llm-verifier-xxx -- curl -v http://api.llm-verifier.com/health

# 7. Fix common issues
case "$(kubectl get pods -l app=llm-verifier --no-headers | grep -E "(CrashLoopBackOff|Error|ImagePullBackOff|Pending)" | wc -l)" in
    1) echo "Pod crash detected - restarting" && kubectl delete pod -l app=llm-verifier --force
    2) echo "Image pull error - checking registry" && kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","imagePullPolicy":"Always"}]}}}}'
    3) echo "Pending pods - checking resources" && kubectl top nodes
    *) echo "Unknown issue - collecting diagnostics" && kubectl exec -it deployment/llm-verifier-xxx -- df -h
esac
```

### Database Connection Issues

```bash
# 1. Check database pod
kubectl get pods -l app=postgres -o wide
kubectl describe pod -l app=postgres

# 2. Test database connectivity
kubectl exec -it postgres-xxx -- pg_isready -U llm_user -d llm_verifier

# 3. Check database logs
kubectl logs postgres-xxx --tail=50

# 4. Verify database configuration
kubectl get secret postgres-secret -o yaml
kubectl get configmap postgres-config -o yaml

# 5. Reset database if needed
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "DROP TABLE IF EXISTS temp_table;"

# 6. Check network policies
kubectl get networkpolicy -n llm-verifier
kubectl describe networkpolicy postgres-netpol
```

### Performance Issues

```bash
# 1. Check resource usage
kubectl top nodes
kubectl top pods -l app=llm-verifier

# 2. Check resource limits
kubectl describe pod -l app=llm-verifier | grep -A "Requests\|Limits"

# 3. Check HPA status
kubectl get hpa llm-verifier-hpa -o yaml
kubectl describe hpa llm-verifier-hpa

# 4. Profile application
kubectl exec -it deployment/llm-verifier-xxx -- curl -s http://localhost:8080/debug/pprof/profile?seconds=30 > /tmp/profile.pprof

# 5. Check database performance
kubectl exec -it postgres-xxx -- psql -U llm_user -d llm_verifier -c "SELECT * FROM pg_stat_activity WHERE state = 'active' ORDER BY query_start DESC LIMIT 5;"

# 6. Network latency testing
kubectl exec -it deployment/llm-verifier-xxx -- ping -c 5 api.llm-verifier.com

# 7. Apply performance fixes
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","resources":{"requests":{"memory":"2Gi","cpu":"1000m"}}]}}}}'
```

### Security Issues

```bash
# 1. Check for security events
kubectl get events -n llm-verifier --field-selector=type=Warning --since=24h

# 2. Check security policies
kubectl get psp -n llm-verifier
kubectl get podsecuritypolicy -n llm-verifier

# 3. Verify secrets are encrypted
kubectl get secret llm-verifier-secrets -o yaml | grep -q "encrypted"

# 4. Check RBAC configuration
kubectl get role -n llm-verifier
kubectl get rolebinding -n llm-verifier
kubectl auth can-i --list --as=system:serviceaccount:llm-verifier

# 5. Scan for vulnerabilities
trivy image --severity HIGH,CRITICAL llm-verifier:v1.0.0
trivy fs --security-checks vuln /app/data

# 6. Check SSL certificates
kubectl get certificate llm-verifier-tls -o yaml
kubectl describe certificate llm-verifier-tls

# 7. Audit file permissions
kubectl exec -it deployment/llm-verifier-xxx -- find /app/data -type f -perm +600
kubectl exec -it deployment/llm-verifier-xxx -- ls -la /app/ssl/

# 8. Apply security fixes
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","securityContext":{"runAsNonRoot":true,"readOnlyRootFilesystem":true}}]}}}}'
```

---

## 📋 Additional Resources

### Security Scripts
- `security-incident-response.sh` - Automated incident response
- `compliance-audit.sh` - SOC 2 and GDPR compliance checking
- `gdpr-compliance-check.sh` - Data privacy compliance

### Monitoring Scripts  
- `health-monitoring.sh` - Continuous health checking
- `performance-analysis.sh` - Automated performance analysis
- `log-analysis.sh` - Intelligent log analysis

### Deployment Scripts
- `blue-green-deploy.sh` - Zero-downtime deployments
- `canary-deployment.sh` - Gradual rollout testing
- `rolling-update.sh` - Safe rolling updates

### Troubleshooting Scripts
- `pod-troubleshooting.sh` - Pod issue diagnosis
- `network-troubleshooting.sh` - Connectivity analysis
- `database-troubleshooting.sh` - Database issue resolution

---

## 🎯 Command Reference

### Quick Commands
```bash
# Health check
kubectl get pods -l app=llm-verifier
curl -f http://api.llm-verifier.com/health

# View logs
kubectl logs -f deployment/llm-verifier --tail=50

# Scale application
kubectl scale deployment llm-verifier --replicas=5

# Rolling update
kubectl set image deployment/llm-verifier llm-verifier:v1.0.1
kubectl rollout status deployment/llm-verifier

# Get metrics
kubectl exec -it deployment/llm-verifier-xxx -- curl -s http://localhost:8080/metrics
```

### Advanced Commands
```bash
# Cluster status overview
kubectl cluster-info && kubectl get nodes -o wide && kubectl get pv && kubectl get pvc

# Full system diagnosis
kubectl get all --all-namespaces -o wide && kubectl top nodes --no-headers && kubectl describe nodes

# Performance analysis
kubectl top pods -l app=llm-verifier -o wide --sort-by=cpu && kubectl describe pods -l app=llm-verifier

# Security audit
kubectl auth can-i --list && kubectl get events --field-selector=type=Warning --since=24h && kubectl get networkpolicy -n llm-verifier

# Resource optimization
kubectl patch deployment llm-verifier -p '{"spec":{"template":{"spec":{"containers":[{"name":"llm-verifier","resources":{"limits":{"memory":"4Gi","cpu":"2000m"}}]}}}}' && kubectl patch hpa llm-verifier-hpa -p '{"spec":{"maxReplicas":10}}'
```

---

**📞 Documentation Maintenance:** This runbook should be reviewed and updated quarterly with new procedures and lessons learned from incidents.

---

**Enhanced LLM Verifier v1.0.0** - Production Operations Guide