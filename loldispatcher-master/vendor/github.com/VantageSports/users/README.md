# users

[![wercker status](https://app.wercker.com/status/3662821d0afddd15cec2b8f450cfe611/m "wercker status")](https://app.wercker.com/project/bykey/3662821d0afddd15cec2b8f450cfe611)

Users API

API that provides user management and administration.

## API Docs

### RPC Endpoints

TODO: Update

Method   | Endpoint                          | Description                 | Data Format
|--------|-----------------------------------|-----------------------------|-----------------
| POST   | /users/v1/login                   | Login                       |
|        |                                   | JSON Body in Request        | {u:username, p:password}
| HEAD   | /users/v1/check                   | Check Token                 |
|        |                                   | Header                      | Authorization: bearer `<token>`
| POST   | /users/v1/changelogin/{userId}    | Change login                | {u:username, p:password}
|        |                                   | Header (temp or from login) | Authorization: bearer `<token>`
| PUT    | /users/v1/forgot_password/{email} | Send Forgot Password link   |

### REST Endpoints

Resource  | Method   | Endpoint    | Description | Query Parameters
:--------:|----------|-------------|-------------|-----------------
User      |          |             |             |
          | POST     | /users/v1/users | Create User
          | GET      | /users/v1/users  | List Users | id, group, role, privilege
          | GET      | /users/v1/user/{id}  | List Single User by id
          | GET      | /users/v1/users/{id}/abilities  | List Abilities for User
          | PUT      | /users/v1/users/{id}  | Update User
          | DELETE   | /users/v1/users/{id}   | Delete User
Group     |          |             |             |
          | POST     | /users/v1/groups | Create Group
          | GET      | /users/v1/groups | List Groups | id, name, role, privilege
          | GET      | /users/v1/groups/{id} | List Single Group by id
          | GET      | /users/v1/groups/{id}/abilities  | List Abilities for Group
          | PUT      | /users/v1/groups/{id}  | Update Group
          | DELETE   | /users/v1/groups/{id}   | Delete Group
Role      |          |             |             |
          | POST     | /users/v1/roles | Create Role
          | GET      | /users/v1/roles | List Roles | id, name, privilege
          | GET      | /users/v1/roles/{id} | List Single Role by id
          | GET      | /users/v1/roles/{id}/abilities  | List Abilities for Role
          | PUT      | /users/v1/roles/{id}  | Update Role
          | DELETE   | /users/v1/roles/{id}   | Delete Role
Privilege |          |             |             |
          | POST     | /users/v1/privileges | Create Privilege
          | GET      | /users/v1/privileges | List Privileges
          | GET      | /users/v1/privileges/{id} | List Single Privilege by id
          | PUT      | /users/v1/privileges/{id}  | Update Privilege
          | DELETE   | /users/v1/privileges/{id}   | Delete Privilege

## Developing

Use docker-compose (on whatever platform you're using) to start the container.

If the SEED env variable is true, it will print out a seed auth token in the logs when it starts up. Copy that token into a file `/tmp/seedtoken` and then either use the seed_dev.sh script (in cli/) to add some initial data to your api server (from testdata.json). OR you can run the cli yourself and interactively add items one at a time from the command line.

## Change Login - Updating User's Email and Password

#### Change Password for New User or User Forgot Password
For a created user or a user that requests `PUT /forgot_password/{userId}`, a temporary auth token is sent to their email. Then the link directs them to the users web UI.

![Sequence Diagram for Password Reset, No Login](http://www.websequencediagrams.com/cgi-bin/cdraw?lz=dGl0bGUgRm9yZ290IFBhc3N3b3JkLAABCSBSZXNldAoKY2xpZW50LT4vZgAlBV9wACMHL3t1c2VySWR9OiBQVVQKAAYZLT50YXNrcy9lbWFpbDogc2VuZCAABwUKbm90ZSBsZWZ0IG9mIAAYDVRlbXBBdXRoVG9rZW4gdG8AKwYgb2YgdXNlcgoARgsAex0yMDAgT0sAgH8cAIFXBgAhCQCBDQZvdmVyIAASCFVzZXIgRGlyZWN0ZWQgdG8gVUkAggkKY2hhbmdlbG9naW4AggMMT1NUAIFXBnJpZ2gAgVkFAII_BgogICB7dTp1c2VybmFtZSwgcDpuZXcgcHdkfQogICBBdXRob3JpemF0aW9uOiBiZWFyZXIAgX4OCmVuZCBub3RlCgoAaBUAgUASAIM2CQCBIQUAgQIbOgCBAhkANQYAgiMKQXV0aCAAgQkGCg&s=default)

#### Change Password for User after Login, Has Auth Token
After a user gets an auth token from `POST /login`, they use that auth token to make a request to `POST /changelogin/{userId}` with a `LoginRequest` JSON body with their current email and new password.

![Sequence Diagram of Authenticated User Change Password](http://www.websequencediagrams.com/cgi-bin/cdraw?lz=dGl0bGUgQXV0aGVudGljYXRlZCBVc2VyIC0gQ2hhbmdlIFBhc3N3b3JkCgpjbGllbnQtPi9sb2dpbjogUE9TVApub3RlIHJpZ2h0IG9mIAAdBjoge3U6dXNlci1lbWFpbCwgcDpwd2R9CgAzBi0-AB4IQXV0aFRva2VuAFEKYwBwBQBcBS97dXNlcklkfQBPGwogIABaEm5ldyAAawUgIACBVQVvcml6YXRpb246IGJlYXJlcgBuC2VuZCBub3RlCgoAZhUAgSIKMjAwIE9LAIFHPgCBEAkAgXUUIACCBAYK&s=default)

#### Change Email for User after Login, Has Auth Token
After a user gets an auth token from `POST /login`, they use that auth token to make a request to `POST /changelogin/{userId}` with a `LoginRequest` JSON body with their new email and password. A temporary auth token is sent to their email. Then the link directs them to the users web UI. Once the web UI sends the request `POST /changelogin/{userId}` with the temporary auth token and updated `LoginRequest` with the new email and current password.

Then the user's email is updated as long as that email is not in use.

![Sequence Diagram of Authenticated User Change Email](http://www.websequencediagrams.com/cgi-bin/cdraw?lz=dGl0bGUgQXV0aGVudGljYXRlZCBVc2VyIC0gQ2hhbmdlIEVtYWlsCgpjbGllbnQtPi9sb2dpbjogUE9TVApub3RlIHJpZ2h0IG9mIAAdBjoge3U6dXNlci1lbWFpbCwgcDpwd2R9CgAzBi0-AB4IQXV0aFRva2VuAFEKYwBtBQBcBS97dXNlcklkfQBPGwogIHt1Om5ldwBbDyAAgUsFb3JpemF0aW9uOiBiZWFyZXIAZwtlbmQgbm90ZQoKAF8VLT50YXNrcy8AgTwFOiBzZW5kIACBSAUAgWsGbGVmAIFrBQAYDVRlbXAAgUoJIHRvACsGIG9mIHVzZXIKAEYLAIFQGTIwMCBPSwB7GACCVggAHwcAgnYGb3ZlcgCCBAoAgzQGRGlyZWN0ZWQgdG8gVUkKICAgbGluayBmcm9tAIE_BwCBdQoAghNoAIIRDQCCdQoAgmcXAINyF3VwZGF0ZXMAgkwFAIMJBwCCBScAhQ8uAIRAEgCFIho&s=default)
