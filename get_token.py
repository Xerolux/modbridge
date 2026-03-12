import requests

r = requests.post('http://localhost:8080/api/login', json={"password": "4iX7HrAXBdkyD6S0"})
print(r.text)
print(r.cookies)
