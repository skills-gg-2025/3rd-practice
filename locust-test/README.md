# Locust
## Docker
```bash
docker run -p 8089:8089 -v $PWD:/mnt/locust locustio/locust -f /mnt/locust/locustfile.py
```
## Execute
```bash
locust -f locustfile.py --csv example --headless -t 10m -u 1000 -r 10 -H $ENDPOINT
```