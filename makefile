setup: 
	go build -o run/out
	docker build run/ -t server
	clear
	docker run -it -p 8080:8080 server



