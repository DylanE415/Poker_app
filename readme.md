compile and run the server with:
- cd server
- go run -race . (use -race for testing to detect data races)
- go to http://localhost:8080/login


to run test fucntions run 
cd server
go test -v -run 'function name'

test besthand logic: go test -v -run TestGetPlayerBestHand_AllTypes


to register a new user
- cd users
- go run 
- copy the hash and put it into the users.json file

*important notes*
- card ranks are 2-14
- suits are "H", "C", "S", "D"


credits for card graphics go to:

Vector Playing Cards 3.2
https://totalnonsense.com/open-source-vector-playing-cards/
Copyright 2011,2021 – Chris Aguilar – conjurenation@gmail.com
Licensed under: LGPL 3.0 - https://www.gnu.org/licenses/lgpl-3.0.html

