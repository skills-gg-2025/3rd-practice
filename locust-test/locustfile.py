from locust import FastHttpUser, task, events
from faker import Faker
import pandas as pd
import threading
import boto3
import os
import time
import uuid
import random

fake = Faker()

user = pd.read_csv('data/users.csv')
emails = user["email"].tolist()

eventId = os.getenv("eventId")
username = os.getenv("username")

user_write_idx = 500000
user_read_idx = 0
product_write_idx = 500000
product_read_idx = random.choices(range(10001, 500000), 10)
index_lock = threading.Lock()

instance_list = []
ec2 = boto3.client("ec2", aws_access_key_id=os.getenv("access_key"), aws_secret_access_key=os.getenv("secret_access_key"), region_name="ap-northeast-2")
ssm = boto3.client("ssm")

def get_hosts_from_ssm(parameter_name):
    response = ssm.get_parameter(Name=parameter_name)
    return response["Parameter"]["Value"]

SHARED_HOST = get_hosts_from_ssm(f"/{eventId}/{username}")

def update_hosts_periodically():
    global SHARED_HOST
    while True:
        time.sleep(60)
        SHARED_HOST = get_hosts_from_ssm(f"/{eventId}/{username}")

threading.Thread(target=update_hosts_periodically, daemon=True).start()

def get_instance_count():
    response = ec2.describe_instances(Filters=[{"Name": "instance-state-name", "Values": ["running"]}])
    instances = sum(len(reservation["Instances"]) for reservation in response["Reservations"])
    return instances

class MonitorThread(threading.Thread):
    def __init__(self, interval=300):
        super().__init__()
        self.interval = interval
        self.running = True
    
    def run(self):
        while self.running:
            instance_count = get_instance_count()
            print(f"[Monitor] Running EC2 Instances: {instance_count}")
            instance_list.append((instance_count,))
            time.sleep(self.interval)

    def stop(self):
        self.running = False

monitor_thread = None

@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    global monitor_thread
    monitor_thread = MonitorThread()
    monitor_thread.start()
    print("[Monitor] EC2 Instance Monitoring Started.")

@events.quitting.add_listener
def on_quitting(environment, **kwargs):
    global monitor_thread
    if monitor_thread:
        monitor_thread.stop()
        monitor_thread.join()
        print("[Monitor] EC2 Instance Monitoring Stopped.")

    df = pd.DataFrame.from_records(instance_list)
    df.to_excel("instances.xlsx")

    s3 = boto3.client("s3")
    metrics = ["exceptions", "failures", "stats_history", "stats"]
    username = os.getenv("username")

    for metric in metrics:
        csv = f"{username}_{metric}.csv"
        s3.upload_file(csv, "student-monitoring", f"{username}/{csv}")
    s3.upload_file("instances.xlsx", "student-monitoring", f"{username}/instances.xlsx")

class TestUser(FastHttpUser):
    @task(2)
    def stress(self):
        self.client.post(SHARED_HOST + "/v1/stress",
                         json={"length": 256, "requestid": "world", "uuid": "skills"}, 
                         name="/v1/stress")

    @task(2)
    def write_user(self):
        global user_write_idx
        with index_lock:
            write_now = user_write_idx
            user_write_idx += 1

        username = fake.unique.user_name()
        email = fake.unique.email()
        status_message = fake.sentence()

        self.client.post(SHARED_HOST + "/v1/user", json={
            "requestid": write_now,
            "uuid": uuid.uuid1(),
            "username": username,
            "email": email,
            "status_message": status_message
        }, name="/v1/user")

        emails.append(email)

    @task(2)
    def read_user(self):
        global user_read_idx
        with index_lock:
            read_now = user_read_idx
            user_read_idx += 1

        self.client.get(SHARED_HOST + f"/v1/user?email={emails[read_now]}&requestid={read_now}&uuid={uuid.uuid1()}", name="/v1/user")

    @task(1)
    def read_user_email_error(self):
        global user_read_idx
        with index_lock:
            read_now = user_read_idx
            user_read_idx += 1

        self.client.get(SHARED_HOST + f"/v1/user?email={emails[read_now].split('.')[0]}&requestid={read_now}&uuid={uuid.uuid1()}", name="error_email")
    
    @task(2)
    def read_product(self):
        read_now = random.choice(product_read_idx)
        self.client.get(SHARED_HOST + f"/v1/product?id={read_now}&requestid={read_now}&uuid={uuid.uuid1()}", name="/v1/product")
    
    @task(2)
    def write_product(self):
        global product_write_idx
        with index_lock:
            write_now = product_write_idx
            product_write_idx += 1

        name = fake.unique.user_name()
        price = random.randint(10000, 50000)

        self.client.post(SHARED_HOST + "/v1/product", json={
            "requestid": write_now,
            "uuid": uuid.uuid1(),
            "id": write_now,
            "name": name,
            "price": price
        }, name="/v1/product")