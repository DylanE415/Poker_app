compile and run the server with:
- cd server
- go run .

for testing, 
run: 
- npx serve 
and open the html file


to run test fucntions run 
cd server
go test -v -run 'function name'

test besthand logic: go test -v -run TestGetPlayerBestHand_AllTypes


to register a new user
- cd users
- go run .
- copy the hash and put it into the users.json file

*important notes*
- card ranks are 2-14
- suits are "H", "C", "S", "D"


