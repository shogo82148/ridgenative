# API Gateway V2 example of ridgenative

It is a saying hello example on API Gateway V1.

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

## Run on AWS Lambda with API Gateway

[template.yaml](template.yaml) is [AWS Serverless Application Model template](https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md) file.
You can run this sample on AWS with API gateway proxy integration.

```
$ ./deploy.sh $YOUR_S3_BUCKET_NAME $YOUR_STACK_NAME
```

```
$ curl -F name=shogo https://$API_ID.lambda-url.$REGION.on.aws/hello
Hello shogo
```
