# AWSMonitor

This is a rough POC CLI tool to pull metadata about your ec2 instances, along with how much traffic (NetworkIn and NetworkOut) they're experiencing.  In theory you could parse and interpret this data to highlight unneeded AWS spend.

`go run main.go list > instances.csv`
