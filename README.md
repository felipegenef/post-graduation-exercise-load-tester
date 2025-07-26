docker build -t loadtester .
docker run loadtester --url=http://google.com --requests=1000 --concurrency=10