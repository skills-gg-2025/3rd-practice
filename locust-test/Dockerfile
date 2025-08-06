FROM locustio/locust

COPY locustfile.py .
COPY requirements.txt .
COPY data data

RUN pip3 install -r requirements.txt