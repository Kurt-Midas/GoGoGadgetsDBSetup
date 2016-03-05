# GoGoGadgetsDBSetup
Uses Golang to translate fuzzwork's sqlite-latest into MongoDB collections. Currently a pre-requisite for Gadgets-Node. 

#How to use
1.	Install Golang, including setting gopath and adding to PATH
2.	Install MongoDB and start a server
3.	Find the Fuzzworks SDE translations. Download "sqlite-latest.sqlite" to the same directory as builder.go and de-archive if necessary
https://www.fuzzwork.co.uk/dump/
4.	Execute with "go run builder.go"
Sequential runs are safe. The tool deletes existing collections before populating new ones. 

#How to update DB versions
1.	Find the latest version of sqlite-latest.sqlite and download to the same directory as builder.go. Do not change the name.
2.	Execute with "go run builder.go"
The tool deletes existing collections before populating new ones. 
No problems are expected from running an update while Gadgets-Node is active but no testing exists to validate that. 

#Comments
Unimpressive but entertaining. 
As for Go, it makes an enjoyable change from Java and from my light experience I like it. I may eventually build a Gadgets-Go backend for practice and experience. It'll never replace Node but I like the idea of Dockerizing individual backend services for use with a Node frontend / API (better than I like a massive single maven project/subproject Spring API and backend, anyways). 