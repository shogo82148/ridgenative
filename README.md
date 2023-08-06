[![Build Status](https://github.com/shogo82148/ridgenative/workflows/Test/badge.svg)](https://github.com/shogo82148/ridgenative/actions)
[![GoDoc](https://godoc.org/github.com/shogo82148/ridgenative?status.svg)](https://godoc.org/github.com/shogo82148/ridgenative)

# ridgenative

AWS Lambda HTTP Proxy integration event bridge to Go net/http.
[fujiwara/ridge](https://github.com/fujiwara/ridge) is a prior work, but it depends on [Apex](http://apex.run/).
I want same one that only depends on [aws/aws-lambda-go](https://github.com/aws/aws-lambda-go).

## SYNOPSIS

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/hello", handleRoot)
	ridgenative.ListenAndServe(":8080", nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello World")
}
```

### Run Locally

You can run it locally.

```
$ go run main.go
```

```
$ curl http://localhost:8080/hello
Hello World
```

### Amazon API Gateway REST API with HTTP proxy integration

You can run it as an [Amazon API Gateway REST API](https://docs.aws.amazon.com/apigateway/latest/developerguide/apigateway-rest-api.html) without any modification of the source code.
Here is an example of [AWS Serverless Application Model template](https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md) template.
See [the example directory](https://github.com/shogo82148/ridgenative/tree/main/example) to how to deploy it.

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Transform: "AWS::Serverless-2016-10-31"
Description: example of shogo82148/ridgenative
Resources:
  ExampleApi:
    Type: AWS::Serverless::Function
    Properties:
      Handler: example
      Runtime: provided.al2
      Timeout: 30
      CodeUri: dist
      Events:
        Proxy:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: any
```

### Amazon API Gateway HTTP API

You can also run it as an [Amazon API Gateway HTTP API](https://docs.aws.amazon.com/apigateway/latest/developerguide/http-api.html).

```yaml
AWSTemplateFormatVersion: "2010-09-09"
Transform: "AWS::Serverless-2016-10-31"
Description: example of shogo82148/ridgenative
Resources:
  ExampleApi:
    Type: AWS::Serverless::Function
    Properties:
      Handler: example
      Runtime: provided.al2
      Timeout: 30
      CodeUri: dist
      Events:
        ApiEvent:
          Type: HttpApi
```

### Targets of Application Load Balancer

More and more, you can run it as [a target of Application Load Balancer](https://docs.aws.amazon.com/elasticloadbalancing/latest/application/lambda-functions.html).

```yaml
AWSTemplateFormatVersion: "2010-09-09"

Resources:
  Function:
    Type: AWS::Lambda::Function
    Properties:
      Code: dist
      Handler: example
      Role: !GetAtt ExecutionRole.Arn
      Runtime: provided.al2
      Timeout: 30

  ExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Effect: Allow
            Principal:
              Service:
                - lambda.amazonaws.com
            Action:
              - sts:AssumeRole
      Path: "/"
      Policies:
        - PolicyName: CloudWatchLogs
          PolicyDocument:
            Statement:
              - Effect: Allow
                Action:
                  - logs:CreateLogGroup
                  - logs:CreateLogStream
                  - logs:PutLogEvents
                Resource: "*"

  LambdaPermission:
    Type: AWS::Lambda::Permission
    Properties:
      Action: lambda:InvokeFunction
      FunctionName: !Ref Function
      Principal: elasticloadbalancing.amazonaws.com

  LambdaTargetGroup:
    Type: AWS::ElasticLoadBalancingV2::TargetGroup
    Properties:
      TargetType: lambda
      Targets:
        - Id: !Att Function.Arn
# Configure listener rules of ALB to forward to the LambdaTargetGroup.
# ...
```

### Lambda function URLs

More and more, you can run it as [Lambda function URLs](https://docs.aws.amazon.com/lambda/latest/dg/lambda-urls.html).

```yaml
AWSTemplateFormatVersion: "2010-09-09"

Resources:
  Function:
    Type: AWS::Lambda::Function
    Properties:
      Code: dist
      Handler: example
      Role: !GetAtt ExecutionRole.Arn
      Runtime: provided.al2
      Timeout: 30

  LambdaUrls:
    Type: AWS::Lambda::Url
    Properties:
      AuthType: NONE
      TargetFunctionArn: !GetAtt Function.Arn

  ExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Effect: Allow
            Principal:
              Service:
                - lambda.amazonaws.com
            Action:
              - sts:AssumeRole
      Path: "/"
      Policies:
        - PolicyName: CloudWatchLogs
          PolicyDocument:
            Statement:
              - Effect: Allow
                Action:
                  - logs:CreateLogGroup
                  - logs:CreateLogStream
                  - logs:PutLogEvents
                Resource: "*"
```

### Lambda function URLs with a Response Streaming Enabled Function

The ridgenative also works with [a response streaming enabled function](https://docs.aws.amazon.com/lambda/latest/dg/configuration-response-streaming.html#config-rs-invoke-furls).
To enable response streaming, set the `RIDGENATIVE_INVOKE_MODE` environment value to `RESPONSE_STREAM`.

```yaml
AWSTemplateFormatVersion: "2010-09-09"

Resources:
  Function:
    Type: AWS::Lambda::Function
    Properties:
      Code: dist
      Handler: example
      Role: !GetAtt ExecutionRole.Arn
      Runtime: provided.al2
      Timeout: 30

      # configure environment values to enable response streaming
      Environment:
        Variables:
          RIDGENATIVE_INVOKE_MODE: RESPONSE_STREAM

  LambdaUrls:
    Type: AWS::Lambda::Url
    Properties:
      AuthType: NONE
      TargetFunctionArn: !GetAtt Function.Arn
      InvokeMode: RESPONSE_STREAM # enables response streaming

  ExecutionRole:
    Type: AWS::IAM::Role
    Properties:
      AssumeRolePolicyDocument:
        Statement:
          - Effect: Allow
            Principal:
              Service:
                - lambda.amazonaws.com
            Action:
              - sts:AssumeRole
      Path: "/"
      Policies:
        - PolicyName: CloudWatchLogs
          PolicyDocument:
            Statement:
              - Effect: Allow
                Action:
                  - logs:CreateLogGroup
                  - logs:CreateLogStream
                  - logs:PutLogEvents
                Resource: "*"
```

With a response streaming enabled function, the ResponseWriter implements `http.Flusher`.

```go
package main

import (
	"fmt"
	"net/http"

	"github.com/shogo82148/ridgenative"
)

func main() {
	http.HandleFunc("/hello", handleRoot)
	ridgenative.ListenAndServe(":8080", nil)
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "Hello World")

	f := w.(http.Flusher)
  f.Flush()
}
```

## RELATED WORKS

- [fujiwara/ridge](https://github.com/fujiwara/ridge)
- [apex/gateway](https://github.com/apex/gateway)
