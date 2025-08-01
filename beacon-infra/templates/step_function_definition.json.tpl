{
  "Comment": "Beacon data ingestion orchestrator using JSONata",
  "QueryLanguage": "JSONata",
  "StartAt": "RunScrapersInParallel",
  "States": {
    "RunScrapersInParallel": {
      "Type": "Parallel",
      "Next": "ProcessScrapedData",
      "Branches": [
        {
          "StartAt": "InvokeGitHubScraper",
          "States": {
            "InvokeGitHubScraper": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${github_scraper_arn}",
                "Payload": "{% $states.input %}"
              },
              "End": true
            }
          }
        },
        {
          "StartAt": "InvokeDatadogScraper",
          "States": {
            "InvokeDatadogScraper": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${datadog_scraper_arn}",
                "Payload": "{% $states.input %}"
              },
              "End": true
            }
          }
        },
        {
          "StartAt": "InvokeAwsScraper",
          "States": {
            "InvokeAwsScraper": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${aws_scraper_arn}",
                "Payload": "{% $states.input %}"
              },
              "End": true
            }
          }
        },
        {
          "StartAt": "InvokeCodeownersScraper",
          "States": {
            "InvokeCodeownersScraper": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${codeowners_scraper_arn}",
                "Payload": "{% $states.input %}"
              },
              "End": true
            }
          }
        },
        {
          "StartAt": "InvokeOpenShiftScraper",
          "States": {
            "InvokeOpenShiftScraper": {
              "Type": "Task",
              "Resource": "arn:aws:states:::lambda:invoke",
              "Parameters": {
                "FunctionName": "${openshift_scraper_arn}",
                "Payload": "{% $states.input %}"
              },
              "End": true
            }
          }
        }
      ],
      "ResultPath": "$.scraper_outputs"
    },
    "ProcessScrapedData": {
      "Type": "Task",
      "Resource": "arn:aws:states:::lambda:invoke",
      "Parameters": {
        "FunctionName": "${processor_arn}",
        "Payload": "{% $states.input.scraper_outputs %}"
      },
      "End": true
    }
  }
}