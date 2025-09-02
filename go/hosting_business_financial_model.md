# Hosting Business Model: AWS Metal + Firecracker

## Core Strategy
- **Hardware**: AWS R6I Metal - 128 vCPUs, 1024 GB RAM
- **Platform Overhead**: 10% reserved for management/monitoring
- **CPU Oversubscription**: 8x (922 sellable vCPUs after overhead)
- **Memory Oversubscription**: None (922 GB sellable after overhead)
- **Virtualization**: Firecracker microVMs

## Pricing Model

**Rate Structure:**
- **Active CPU**: $0.015/vCPU-hour (pay for actual usage)
- **Provisioned Memory**: $0.015/GB-hour (pay while provisioned)

**Formula**: `hours × (active_cpus × $0.015 + provisioned_gb × $0.015)`

**Example (4 vCPU + 8 GB):**
- High load (3 vCPUs active): $10.80/10 days
- Low usage (0.1 vCPUs active): $0.18/5 days
- Normal ops (1.5 vCPUs active): $8.10/15 days
- Memory: 8 GB × 720h × $0.015 = $86.40/month
- **Total: $105.48/month**
- **TODO**: Verify competitive pricing against Railway, Vercel, Render

## Target: 2.5x Hardware Cost Multiplier

| Pricing Model | Monthly Cost | 2.5x Target Revenue | Break-even Customers* | Max Customers** | Feasible? |
|---------------|--------------|---------------------|----------------------|----------------|-----------|
| **On-Demand** | $5,887 | $14,718 | Auto-achieved ✅ | 230 customers | ✅ **Always profitable** |
| **1-Year Reserved** | $3,894 | $9,735 | Auto-achieved ✅ | 230 customers | ✅ **Always profitable** |

*With cost-based pricing, we automatically achieve 2.5x markup on every customer
**Maximum customers assuming average 4GB provisioned memory per customer (922GB ÷ 4GB)

## Customer Configuration Analysis

**How pricing works for different customer types:**
- **vCPUs**: Number of virtual CPUs provisioned
- **Memory (GB)**: Amount of memory provisioned
- **Avg CPU Usage**: Percentage of provisioned vCPUs actually running (30% = busy 30% of the time)
- **Monthly Cost**: Total bill = (CPU usage × hours × $0.015) + (memory provisioned × hours × $0.015)

| vCPUs | Memory (GB) | Avg CPU Usage | Monthly Cost | Use Case |
|-------|-------------|---------------|--------------|----------|
| 1 | 1 | 20% | $13.32 | Small API, dev environment |
| 2 | 2 | 30% | $28.08 | Medium API, staging |
| 4 | 4 | 40% | $60.48 | Production API, web app |
| 8 | 8 | 50% | $129.60 | High-traffic API, background jobs |
| 16 | 16 | 60% | $276.48 | Enterprise API, data processing |
| 2 | 8 | 15% | $88.56 | Memory-heavy cache, database |
| 8 | 4 | 70% | $117.36 | CPU-intensive processing |

**Pricing scales linearly with resource usage**

## Competitive Analysis
**TODO**: Research current pricing for Railway, Vercel, Render, Fly.io
- Compare 4 vCPU + 8GB configurations
- Include flat-rate vs usage-based models
- Calculate savings for typical API workloads
