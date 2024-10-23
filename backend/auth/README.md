<!--
SPDX-License-Identifier: Apache-2.0
SPDX-FileCopyrightText: 2024 Canonical Ltd
-->

# Authentication and Authorization Feature

Webui is the upstream component that offers an API to configure the 5G core network in SD-Core. With the implementation of the Authentication and Authorization feature, security risks have been reduced, ensuring that access is restricted to authorized users only.

This is an optional feature that is disabled by default.

## The Feature

JWT is used to authenticate users, ensuring secure access to the system. For protected endpoints, users must include a `token`, which Webui uses to verify their identity and grant access.

Authorization is implemented based on these 2 roles:`AdminRole` and `UserRole`. The webui uses the `token` to determine the role of the user performing the action.

Depending on their role, users have different levels of access to Webui operations:

- `UserRole`: Users with this role can retrieve their own account information and change their own password.
- `AdminRole`: Admin users have full access to all endpoints, allowing them to perform any action on their own account as well as on other users' accounts.

Both `AdminRole` and `UserRole` users can manage additional resources, such as Network Slices, Device Groups, and Subscribers, but they must be logged in.

The `AdminRole` user cannot be deleted.

## Setup

### Enable the Feature

To enable this feature, add the following parameters to the config file.
```
configuration:
  enableAuthentication: true
  mongodb:
    . . .
    webuiDbName: <name>
    webuiDbUrl: <url>
```

### First User Creation

On a fresh deployment, the endpoint for creating a new user is not protected, allowing initial setup without authentication:

```
curl -v "localhost:5000/config/v1/account" \
--data '{
 "username": <username>,
 "password": <password>
}'
```

The first user created will automatically be assigned the `AdminRole`. Only one user can hold the `AdminRole`, and this user cannot be deleted.

For all subsequent user creations, authentication is required by logging in.

## Endpoints that does not require authorization

There are two endpoints that can be accessed without providing a JWT token in the request header.

### Log in

The log in operation is required before performing most actions on the Webui. It generates a `token`, which must be included in subsequent requests for authentication.

```
curl -v "localhost:5000/login" \
--header 'Content-Type: application/json' \
--data '{
  "username": <username>,
  "password": <password>
}'
```
Response:
```
{"token":"eyJhbG123aad1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VybmFtZSI6ImFkbWluVXNlciIsInBlcm1pc3Npb25zIjoxLCJleHAiOjE3MjY1ODIyNTZ9.YU6tveV3oXcfGMvqB7xIcP1Fs6c6ZZoP134Y8ozV4lA"}
```

### Get Status
This endpoint allows any user to check the status of the Webui without a `token`. The status indicates whether the system's initialization has occurred, specifically whether the first `AdminRole` user has been created, and if the system is ready for use.

```
curl -v "localhost:5000/status"
```
Response:
```
{"initialized":false}
```
or
```
{"initialized":true}
```

### First User Creation

As mentioned above, the [First User Creation](#first-user-creation) endpoint does not require a `JWT token`, allowing initial setup without authentication.

## User Management Endpoints

To access any of the following endpoints, users must be logged in and include a valid JWT token in the request header.

### Create User
Create a new user by providing the username and password.
```
curl -v -H "Authorization: Bearer <token>" "localhost:5000/config/v1/account" \
--data '{
 "username": <username>,
 "password": <password>
}'

```

### Get Users
Retrieve a list of all users.
```
curl -v -H "Authorization: Bearer <token>" "localhost:5000/config/v1/account" 
```
Response:
```
[{"username":"adminUser","role":1}]
```

### Get User
Retrieve details for a specific user by their username.
```
curl -v -H "Authorization: Bearer <token>" "localhost:5000/config/v1/account/<username>" 
```
Response:
```
{"username":"adminUser","role":1}
```

### Change Password
Change the password for a specific user.
```
curl -v -H "Authorization: Bearer <token>" "localhost:5000/config/v1/account/<username>/change_password" \
--data '{
  "password": <new_password>
}'
```

### Delete User
Delete a specific user by their username.
```
curl -v -H "Authorization: Bearer <token>" -X DELETE  "localhost:5000/v1/config/account/<username>" 
```

## Othe Endpoints

Configuration endpoints now require the inclusion of a JWT token in the request header for authorization.
``` 
curl -v -H "Authorization: Bearer <token>" "localhost:5000/api/subscriber/<imsi>" \
--header 'Content-Type: text/plain' \
--data '{
    ...
}'
```