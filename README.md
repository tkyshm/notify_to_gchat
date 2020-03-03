# Cloud Function to alert notification to Hangouts Chat from Stackdriver webhook notification.

## deploy

```
$ gcloud functions deploy notify_to_gchat \
    --entry-point NotifyToGChat \
    --runtime go111 \
    --set-env-vars 'WEBHOOK_URL=...' \
    --trigger-http \
    --project ... \
    --region ...
```
