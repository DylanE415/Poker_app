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


to make build folder run build.sh(builds a binary for linux x86)
./build.sh

FOR SETTING APP ON AN EC2:
- to ssh into ec2 instance
ssh -i ~/.ssh/poker_key.pem ec2-user@54.183.19.220
- to copy build folder into ec2 into an app folder (from inside /poker_app)
scp -i ~/.ssh/my-ec2-key.pem -r build/* ec2-user@54.183.19.220:/home/ec2-user/app/
- allow binary to bind to port 80
sudo setcap 'cap_net_bind_service=+ep' /home/ec2-user/app/server/server
- make symlink for users folder
ln -s /home/ec2-user/app/users /home/ec2-user/users

 -run binary
chmod +x server/server
./server/server




*important notes*
- card ranks are 2-14
- suits are "H", "C", "S", "D"


credits for card graphics go to:

Vector Playing Cards 3.2
https://totalnonsense.com/open-source-vector-playing-cards/
Copyright 2011,2021 – Chris Aguilar – conjurenation@gmail.com
Licensed under: LGPL 3.0 - https://www.gnu.org/licenses/lgpl-3.0.html

