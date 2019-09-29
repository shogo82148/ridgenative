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

You can run it locally.

```
$ go run main.go
```

```
$ curl http://localhost:8080/hello
Hello World
```

And more, you can run it on AWS with API gateway proxy integration.
Here is an example of [AWS Serverless Application Model template](https://github.com/awslabs/serverless-application-model/blob/master/versions/2016-10-31.md) template.
See [the example directory](https://github.com/shogo82148/ridgenative/tree/master/example) to how to deploy it.

```yaml
AWSTemplateFormatVersion: '2010-09-09'
Transform: 'AWS::Serverless-2016-10-31'
Description: example of shogo82148/ridgenative
Resources:
  ExampleApi:
    Type: AWS::Serverless::Function
    Properties:
      Handler: example
      Runtime: go1.x
      Timeout: 30
      CodeUri: dist
      Events:
        Proxy:
          Type: Api
          Properties:
            Path: /{proxy+}
            Method: any
```

More and more, you can run it on AWS with Application Load Balancer.

```yaml
AWSTemplateFormatVersion: "2010-09-09"

Resources:
  Function:
    Type: AWS::Lambda::Function
    DependsOn: ExecutionRole
    Properties:
      Code: dist
      Handler: example
      Role: !GetAtt ExecutionRole.Arn
      Runtime: go1.x
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

## RELATED WORKS

- [fujiwara/ridge](https://github.com/fujiwara/ridge)
- [apex/gateway](https://github.com/apex/gateway)
