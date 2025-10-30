compile and run the server with:
- cd server
- go run -race . (use -race for testing to detect data races)
- go to http://localhost:8080/login


to run _test.go files run 
cd server
go test -v 

test besthand logic: go test -v -run TestGetPlayerBestHand_AllTypes


to register a new user
- cd users
- go run 
- copy the hash and put it into the users.json file


- to make a new build folder run build.py(builds a binary for linux x86):
build/
  server/server   # linux go binary
  static/...
  users/...

- then run chmod +x build/server/server BEFORE BUILDING DOCKER IMAGE



# to auto deploy onto your ec2 instance:
    docker image inspect poker:latest >/dev/null 2>&1 && docker rmi poker:latest; docker buildx build --platform linux/amd64 -t poker:latest .


# ssh into your ec2 to install and run docker:
    ssh -i ~/.ssh/poker_key.pem ec2-user@54.183.19.220
    sudo dnf install -y docker 
    sudo systemctl start docker
    sudo systemctl enable docker

# Load image into ec2
docker save poker:latest | bzip2 | \
  ssh -i ~/.ssh/poker_key.pem ec2-user@54.183.19.220 \
  'sudo docker rmi -f poker:latest 2>/dev/null || true && bunzip2 | sudo docker load'


# run image
ssh -i ~/.ssh/poker_key.pem ec2-user@54.183.19.220 \
  'sudo docker rm -f poker 2>/dev/null || true && sudo docker run -d --name poker --restart=always -p 80:8080 poker:latest'


# See if it is working 

ssh -i ~/.ssh/poker_key.pem ec2-user@54.183.19.220
sudo docker ps






*important notes*
- card ranks are 2-14
- suits are "H", "C", "S", "D"


credits for card graphics go to:

Vector Playing Cards 3.2
https://totalnonsense.com/open-source-vector-playing-cards/
Copyright 2011,2021 – Chris Aguilar – conjurenation@gmail.com
Licensed under: LGPL 3.0 - https://www.gnu.org/licenses/lgpl-3.0.html



