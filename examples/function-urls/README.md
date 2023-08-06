# Lambda Function URLs example of ridgenative

It is a saying hello example on Lambda Function URLs.

```
$ curl -F name=shogo https://xxxxxxx.lambda-url.ap-northeast-1.on.aws/hello
```

## RUN LOCALLY

```
$ go run main.go
```

```
$ curl -F name=shogo http://localhost:8080/hello
Hello shogo
```

## Run on AWS Lambda

[template.yaml](template.yaml) is [AWS Serverless Application Model template](https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md) file.
You can run this sample on AWS with Lambda Function URLs.

```
$ ./deploy.sh $YOUR_S3_BUCKET_NAME $YOUR_STACK_NAME
```

```
$ curl -F name=shogo https://$API_ID.lambda-url.$REGION.on.aws/hello
Hello shogo
```
