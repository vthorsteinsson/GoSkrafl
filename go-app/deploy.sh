
# Deploy the GoSkrafl server to Google App Engine
if [[ $1 == "" ]]; then
    echo "Version is missing. Usage: deploy.sh <version>"
    exit 1
fi
gcloud app deploy --project=explo-dev --no-promote --version=$1 --no-cache
