# Google Cloud Shell 一键配置脚本（修复版 - 单行命令）
# 使用方法：
# 1. 先设置项目: gcloud config set project YOUR_PROJECT_ID
# 2. 然后执行下面的命令
# 或者直接在命令中指定项目: PROJECT_ID="your-project-id" bash -c '...'

PROJECT_ID="${PROJECT_ID:-$(gcloud config get-value project 2>/dev/null | grep -v '^$' || echo '')}" && \
if [ -z "$PROJECT_ID" ]; then \
  echo "错误: 未设置项目 ID" && \
  echo "请先运行: gcloud config set project YOUR_PROJECT_ID" && \
  echo "或查看可用项目: gcloud projects list" && \
  exit 1; \
fi && \
echo "项目: $PROJECT_ID" && \
(gcloud compute firewall-rules describe allow-http-80-oneapi --project="$PROJECT_ID" &>/dev/null && \
 gcloud compute firewall-rules update allow-http-80-oneapi --allow tcp:80 --source-ranges 0.0.0.0/0 --description "Allow HTTP traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTP 规则已更新" || \
 gcloud compute firewall-rules create allow-http-80-oneapi --allow tcp:80 --source-ranges 0.0.0.0/0 --description "Allow HTTP traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTP 规则已创建") && \
(gcloud compute firewall-rules describe allow-https-443-oneapi --project="$PROJECT_ID" &>/dev/null && \
 gcloud compute firewall-rules update allow-https-443-oneapi --allow tcp:443 --source-ranges 0.0.0.0/0 --description "Allow HTTPS traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTPS 规则已更新" || \
 gcloud compute firewall-rules create allow-https-443-oneapi --allow tcp:443 --source-ranges 0.0.0.0/0 --description "Allow HTTPS traffic for One-API" --project="$PROJECT_ID" --quiet && \
 echo "✓ HTTPS 规则已创建") && \
echo "" && \
echo "配置完成！查看规则：" && \
gcloud compute firewall-rules list --filter="name~oneapi" --format="table(name,allowed,direction,sourceRanges)" --project="$PROJECT_ID"
