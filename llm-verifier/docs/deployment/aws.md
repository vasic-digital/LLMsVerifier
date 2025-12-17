# AWS Deployment Guide

This guide covers deploying the LLM Verifier application on Amazon Web Services (AWS).

## Prerequisites

- AWS CLI configured
- Terraform 1.0+ (recommended for infrastructure as code)
- AWS account with appropriate permissions
- Domain name (optional, for production)

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   CloudFront    │    │   Application   │    │     RDS/Aurora  │
│   (CDN/Global)  │◄──►│     Load        │◄──►│   (Database)    │
│                 │    │   Balancer      │    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Route 53      │    │     EKS/ECS     │    │     ElastiCache │
│  (DNS)          │    │   (Container)   │    │   (Redis)        │
└─────────────────┘    └─────────────────┘    └─────────────────┘
         ▲                       ▲                       ▲
         │                       │                       │
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Certificate   │    │     S3          │    │     CloudWatch  │
│   Manager       │    │   (Storage)     │    │   (Monitoring)   │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

## Quick Start with CloudFormation

1. **Launch CloudFormation stack:**
   ```bash
   aws cloudformation create-stack \
     --stack-name llm-verifier \
     --template-url https://llm-verifier-templates.s3.amazonaws.com/main.yaml \
     --parameters ParameterKey=Environment,ParameterValue=production
   ```

2. **Wait for completion:**
   ```bash
   aws cloudformation wait stack-create-complete --stack-name llm-verifier
   ```

3. **Get outputs:**
   ```bash
   aws cloudformation describe-stacks --stack-name llm-verifier --query 'Stacks[0].Outputs'
   ```

## Detailed AWS Deployment

### 1. VPC and Networking

```hcl
# Terraform configuration
resource "aws_vpc" "llm_verifier" {
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "llm-verifier-vpc"
  }
}

resource "aws_subnet" "private" {
  count             = 3
  vpc_id            = aws_vpc.llm_verifier.id
  cidr_block        = cidrsubnet(aws_vpc.llm_verifier.cidr_block, 8, count.index)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "llm-verifier-private-${count.index + 1}"
  }
}

resource "aws_subnet" "public" {
  count             = 3
  vpc_id            = aws_vpc.llm_verifier.id
  cidr_block        = cidrsubnet(aws_vpc.llm_verifier.cidr_block, 8, count.index + 3)
  availability_zone = data.aws_availability_zones.available.names[count.index]

  tags = {
    Name = "llm-verifier-public-${count.index + 1}"
  }
}
```

### 2. Security Groups

```hcl
resource "aws_security_group" "alb" {
  name_prefix = "llm-verifier-alb-"
  vpc_id      = aws_vpc.llm_verifier.id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "ecs" {
  name_prefix = "llm-verifier-ecs-"
  vpc_id      = aws_vpc.llm_verifier.id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
```

### 3. Application Load Balancer

```hcl
resource "aws_lb" "llm_verifier" {
  name               = "llm-verifier-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = aws_subnet.public[*].id

  enable_deletion_protection = true

  tags = {
    Name = "llm-verifier-alb"
  }
}

resource "aws_lb_target_group" "llm_verifier" {
  name        = "llm-verifier-tg"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = aws_vpc.llm_verifier.id
  target_type = "ip"

  health_check {
    enabled             = true
    healthy_threshold   = 2
    interval            = 30
    matcher             = "200"
    path                = "/health"
    port                = "traffic-port"
    protocol            = "HTTP"
    timeout             = 5
    unhealthy_threshold = 2
  }
}

resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.llm_verifier.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type = "redirect"

    redirect {
      port        = "443"
      protocol    = "HTTPS"
      status_code = "HTTP_301"
    }
  }
}

resource "aws_lb_listener" "https" {
  load_balancer_arn = aws_lb.llm_verifier.arn
  port              = "443"
  protocol          = "HTTPS"
  ssl_policy        = "ELBSecurityPolicy-2016-08"
  certificate_arn   = aws_acm_certificate.llm_verifier.arn

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.llm_verifier.arn
  }
}
```

### 4. RDS Database

```hcl
resource "aws_db_subnet_group" "llm_verifier" {
  name       = "llm-verifier-db"
  subnet_ids = aws_subnet.private[*].id

  tags = {
    Name = "llm-verifier-db"
  }
}

resource "aws_rds_cluster" "llm_verifier" {
  cluster_identifier     = "llm-verifier"
  engine                 = "aurora-mysql"
  engine_version         = "8.0.mysql_aurora.3.02.0"
  database_name          = "llm_verifier"
  master_username        = "admin"
  master_password        = random_password.db_password.result
  db_subnet_group_name   = aws_db_subnet_group.llm_verifier.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  backup_retention_period = 7
  preferred_backup_window = "03:00-04:00"

  scaling_configuration {
    auto_pause               = true
    max_capacity             = 16
    min_capacity             = 2
    seconds_until_auto_pause = 300
  }

  tags = {
    Name = "llm-verifier-db"
  }
}
```

### 5. ElastiCache (Redis)

```hcl
resource "aws_elasticache_subnet_group" "llm_verifier" {
  name       = "llm-verifier-redis"
  subnet_ids = aws_subnet.private[*].id
}

resource "aws_elasticache_cluster" "llm_verifier" {
  cluster_id           = "llm-verifier-redis"
  engine               = "redis"
  node_type            = "cache.t3.micro"
  num_cache_nodes      = 1
  parameter_group_name = "default.redis6.x"
  port                 = 6379
  subnet_group_name    = aws_elasticache_subnet_group.llm_verifier.name
  security_group_ids   = [aws_security_group.redis.id]

  tags = {
    Name = "llm-verifier-redis"
  }
}
```

### 6. ECS Fargate Deployment

```hcl
resource "aws_ecs_cluster" "llm_verifier" {
  name = "llm-verifier"

  setting {
    name  = "containerInsights"
    value = "enabled"
  }

  tags = {
    Name = "llm-verifier"
  }
}

resource "aws_ecs_task_definition" "llm_verifier" {
  family                   = "llm-verifier"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "1024"
  memory                   = "2048"
  execution_role_arn       = aws_iam_role.ecs_execution.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([
    {
      name  = "llm-verifier"
      image = "${aws_ecr_repository.llm_verifier.repository_url}:latest"

      environment = [
        {
          name  = "DATABASE_URL"
          value = "mysql://${aws_rds_cluster.llm_verifier.master_username}:${random_password.db_password.result}@${aws_rds_cluster.llm_verifier.endpoint}:${aws_rds_cluster.llm_verifier.port}/${aws_rds_cluster.llm_verifier.database_name}"
        },
        {
          name  = "REDIS_URL"
          value = "redis://${aws_elasticache_cluster.llm_verifier.cache_nodes[0].address}:${aws_elasticache_cluster.llm_verifier.cache_nodes[0].port}"
        }
      ]

      secrets = [
        {
          name      = "OPENAI_API_KEY"
          valueFrom = "${aws_secretsmanager_secret.openai.arn}:api_key::"
        },
        {
          name      = "ANTHROPIC_API_KEY"
          valueFrom = "${aws_secretsmanager_secret.anthropic.arn}:api_key::"
        }
      ]

      portMappings = [
        {
          containerPort = 8080
          protocol      = "tcp"
        }
      ]

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = "/ecs/llm-verifier"
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }

      healthCheck = {
        command = ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
        interval = 30
        timeout  = 5
        retries  = 3
      }
    }
  ])

  tags = {
    Name = "llm-verifier"
  }
}

resource "aws_ecs_service" "llm_verifier" {
  name            = "llm-verifier"
  cluster         = aws_ecs_cluster.llm_verifier.id
  task_definition = aws_ecs_task_definition.llm_verifier.arn
  desired_count   = 2

  network_configuration {
    security_groups = [aws_security_group.ecs.id]
    subnets         = aws_subnet.private[*].id
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.llm_verifier.arn
    container_name   = "llm-verifier"
    container_port   = 8080
  }

  deployment_controller {
    type = "ECS"
  }

  lifecycle {
    ignore_changes = [task_definition]
  }

  tags = {
    Name = "llm-verifier"
  }
}
```

## Monitoring and Observability

### CloudWatch Alarms

```hcl
resource "aws_cloudwatch_metric_alarm" "high_cpu" {
  alarm_name          = "llm-verifier-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/ECS"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"
  alarm_description   = "This metric monitors ecs cpu utilization"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  dimensions = {
    ClusterName = aws_ecs_cluster.llm_verifier.name
    ServiceName = aws_ecs_service.llm_verifier.name
  }
}
```

### X-Ray Integration

```hcl
resource "aws_xray_sampling_rule" "llm_verifier" {
  rule_name      = "llm-verifier"
  priority       = 10
  version        = 1
  reservoir_size = 1
  fixed_rate     = 0.05
  url_path       = "*"
  host           = "*"
  http_method    = "*"
  service_type   = "*"
  service_name   = "*"
  resource_arn   = "*"
}
```

## Security

### IAM Roles and Policies

```hcl
resource "aws_iam_role" "ecs_execution" {
  name = "llm-verifier-ecs-execution"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_execution" {
  role       = aws_iam_role.ecs_execution.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

resource "aws_iam_role" "ecs_task" {
  name = "llm-verifier-ecs-task"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ecs-tasks.amazonaws.com"
        }
      }
    ]
  })
}

resource "aws_iam_policy" "ecs_task" {
  name        = "llm-verifier-ecs-task-policy"
  description = "Policy for LLM Verifier ECS task"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "kms:Decrypt"
        ]
        Resource = "*"
      }
    ]
  })
}
```

### Secrets Management

```hcl
resource "aws_secretsmanager_secret" "openai" {
  name                    = "llm-verifier/openai"
  description             = "OpenAI API key for LLM Verifier"
  recovery_window_in_days = 0

  tags = {
    Name = "llm-verifier-openai-key"
  }
}

resource "aws_secretsmanager_secret_version" "openai" {
  secret_id = aws_secretsmanager_secret.openai.id
  secret_string = jsonencode({
    api_key = var.openai_api_key
  })
}
```

### WAF and Shield

```hcl
resource "aws_wafv2_web_acl" "llm_verifier" {
  name        = "llm-verifier-waf"
  description = "WAF for LLM Verifier"
  scope       = "REGIONAL"

  default_action {
    allow {}
  }

  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 1

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        name        = "AWSManagedRulesCommonRuleSet"
        vendor_name = "AWS"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name                = "AWSManagedRulesCommonRuleSet"
      sampled_requests_enabled   = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name                = "llm-verifier-waf"
    sampled_requests_enabled   = true
  }
}
```

## Backup and Disaster Recovery

### Automated Backups

```hcl
resource "aws_backup_plan" "llm_verifier" {
  name = "llm-verifier-backup-plan"

  rule {
    rule_name         = "llm-verifier-daily"
    target_vault_name = aws_backup_vault.llm_verifier.name
    schedule          = "cron(0 5 ? * * *)"

    lifecycle {
      delete_after = 30
    }
  }

  tags = {
    Name = "llm-verifier-backup"
  }
}

resource "aws_backup_selection" "llm_verifier" {
  name         = "llm-verifier-backup-selection"
  plan_id      = aws_backup_plan.llm_verifier.id
  iam_role_arn = aws_iam_role.backup.arn

  resources = [
    aws_rds_cluster.llm_verifier.arn,
    aws_efs_file_system.llm_verifier.arn
  ]
}
```

### Cross-Region Replication

```hcl
resource "aws_s3_bucket_replication_configuration" "llm_verifier" {
  bucket = aws_s3_bucket.llm_verifier.id
  role   = aws_iam_role.replication.arn

  rule {
    id     = "replicate-to-dr-region"
    status = "Enabled"

    destination {
      bucket        = aws_s3_bucket.llm_verifier_dr.arn
      storage_class = "STANDARD"
    }
  }
}
```

## Cost Optimization

### Auto Scaling

```hcl
resource "aws_appautoscaling_target" "llm_verifier" {
  max_capacity       = 10
  min_capacity       = 2
  resource_id        = "service/${aws_ecs_cluster.llm_verifier.name}/${aws_ecs_service.llm_verifier.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

resource "aws_appautoscaling_policy" "cpu" {
  name               = "cpu-autoscaling"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.llm_verifier.resource_id
  scalable_dimension = aws_appautoscaling_target.llm_verifier.scalable_dimension
  service_namespace  = aws_appautoscaling_target.llm_verifier.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value = 70.0
  }
}
```

### Spot Instances (for non-critical workloads)

```hcl
resource "aws_ecs_capacity_provider" "spot" {
  name = "llm-verifier-spot"

  auto_scaling_group_provider {
    auto_scaling_group_arn         = aws_autoscaling_group.spot.arn
    managed_termination_protection = "ENABLED"

    managed_scaling {
      maximum_scaling_step_size = 10
      minimum_scaling_step_size = 1
      status                    = "ENABLED"
      target_capacity           = 100
    }
  }
}
```

## CI/CD Pipeline

### CodePipeline with CodeBuild

```hcl
resource "aws_codepipeline" "llm_verifier" {
  name     = "llm-verifier-pipeline"
  role_arn = aws_iam_role.codepipeline.arn

  artifact_store {
    location = aws_s3_bucket.artifacts.bucket
    type     = "S3"
  }

  stage {
    name = "Source"

    action {
      name             = "Source"
      category         = "Source"
      owner            = "AWS"
      provider         = "CodeCommit"
      version          = "1"
      output_artifacts = ["source_output"]

      configuration = {
        RepositoryName = "llm-verifier"
        BranchName     = "main"
      }
    }
  }

  stage {
    name = "Build"

    action {
      name             = "Build"
      category         = "Build"
      owner            = "AWS"
      provider         = "CodeBuild"
      input_artifacts  = ["source_output"]
      output_artifacts = ["build_output"]
      version          = "1"

      configuration = {
        ProjectName = aws_codebuild_project.llm_verifier.name
      }
    }
  }

  stage {
    name = "Deploy"

    action {
      name            = "Deploy"
      category        = "Deploy"
      owner           = "AWS"
      provider        = "ECS"
      input_artifacts = ["build_output"]
      version         = "1"

      configuration = {
        ClusterName = aws_ecs_cluster.llm_verifier.name
        ServiceName = aws_ecs_service.llm_verifier.name
        FileName    = "imagedefinitions.json"
      }
    }
  }
}
```

## Troubleshooting

### Common Issues

1. **ECS task fails to start:**
   ```bash
   aws ecs describe-tasks --cluster llm-verifier --tasks <task-id>
   aws logs get-log-events --log-group-name /ecs/llm-verifier --log-stream-name ecs/<task-id>
   ```

2. **Database connection issues:**
   ```bash
   aws rds describe-db-clusters --db-cluster-identifier llm-verifier
   aws rds describe-db-cluster-endpoints --db-cluster-identifier llm-verifier
   ```

3. **Load balancer health check failures:**
   ```bash
   aws elbv2 describe-target-health --target-group-arn <target-group-arn>
   ```

### Debugging Commands

```bash
# Check ECS service status
aws ecs describe-services --cluster llm-verifier --services llm-verifier

# Check ALB access logs
aws s3 cp s3://llm-verifier-logs/2024/01/01/ <local-dir> --recursive

# Check CloudWatch metrics
aws cloudwatch get-metric-statistics \
  --namespace AWS/ECS \
  --metric-name CPUUtilization \
  --dimensions Name=ClusterName,Value=llm-verifier Name=ServiceName,Value=llm-verifier \
  --start-time 2024-01-01T00:00:00Z \
  --end-time 2024-01-02T00:00:00Z \
  --period 3600 \
  --statistics Average

# Check X-Ray traces
aws xray get-trace-summaries --start-time 2024-01-01T00:00:00Z --end-time 2024-01-02T00:00:00Z
```

## Performance Optimization

### CloudFront CDN

```hcl
resource "aws_cloudfront_distribution" "llm_verifier" {
  origin {
    domain_name = aws_lb.llm_verifier.dns_name
    origin_id   = "llm-verifier-alb"

    custom_origin_config {
      http_port              = 80
      https_port             = 443
      origin_protocol_policy = "https-only"
      origin_ssl_protocols   = ["TLSv1.2"]
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  default_root_object = ""

  default_cache_behavior {
    allowed_methods  = ["DELETE", "GET", "HEAD", "OPTIONS", "PATCH", "POST", "PUT"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "llm-verifier-alb"

    forwarded_values {
      query_string = true
      cookies {
        forward = "all"
      }
      headers = ["Authorization", "Content-Type"]
    }

    viewer_protocol_policy = "redirect-to-https"
    min_ttl                = 0
    default_ttl            = 3600
    max_ttl                = 86400
  }

  restrictions {
    geo_restriction {
      restriction_type = "none"
    }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate.cloudfront.arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  tags = {
    Name = "llm-verifier-cdn"
  }
}
```

### Database Performance

```hcl
resource "aws_rds_cluster_parameter_group" "llm_verifier" {
  family = "aurora-mysql8.0"
  name   = "llm-verifier-mysql"

  parameter {
    name  = "innodb_buffer_pool_size"
    value = "{DBInstanceClassMemory*3/4}"
  }

  parameter {
    name  = "max_connections"
    value = "1000"
  }

  parameter {
    name  = "query_cache_size"
    value = "268435456"
  }

  parameter {
    name  = "query_cache_type"
    value = "1"
  }
}
```

This AWS deployment provides a production-ready, scalable, and secure infrastructure for the LLM Verifier application with comprehensive monitoring, backup, and disaster recovery capabilities.