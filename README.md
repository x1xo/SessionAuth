# Identity Provider based on Sessions 🔐

I needed a small authentication service, so I created service using JWT's but I still needed to make the system statefull
to invalidate the sessions, so instead i made this service that handles user authentication using sessions. Let's dive into the details! 🚀

## Build and Run 🏃‍♀️

You can build the service by executing the following command:
```go build -o ./build ./src/main.go```


To run the service, use the following command:
```./build```

## Routes 🛣️

### Login 🔑

To initiate the login process, navigate to `/login?provider=<provider>`. Replace `<provider>` with the desired authentication provider such as GitHub, Google, or Discord. This will redirect you to the OAuth screen of the selected provider, where you can authenticate yourself securely.

### Callbacks 🔄

For each authentication provider, you need to add a callback URL. The callback URL should follow this format: `/callback/provider`, where `provider` corresponds to the authentication provider you are integrating (e.g., `/callback/google` for Google authentication). After successful authentication, the provider will redirect the user back to the specified callback URL in the `.env` file: `REDIRECT_URL`.

### User Info 👤

To retrieve user information for a specific session, you must store the session ID as a cookie named `session`. Once the session ID cookie is set, you can navigate to `/api/user` to fetch the user information associated with the session. This endpoint provides a convenient way to access user details after successful authentication.

### User Sessions 📆

This service allows you to manage user sessions effectively. You can view all the active sessions that are currently valid for a particular user. Additionally, you have the option to invalidate specific sessions by blocking their corresponding session ID.

To see all the sessions for a user, simply navigate to `/api/user/sessions/`. This endpoint provides an overview of all the active sessions associated with the user.

If you need to invalidate a session, send a `DELETE` request to `/api/user/sessions/<sessionid>`. This endpoint will invalidate the session with the associated `sessionid` and block future requests with that session ID. When invalidating, your app can also store the `sessionid` to reduce the round trip for checking the validity of a session.

To invalidate all sessions, send a `DELETE` request to `/api/user/sessions/invalidate_all`. This endpoint will invalidate all sessions associated with the current user.

**Note:** When accessing any `/api` routes, make sure to pass the `session` cookie in your request. OR you can also 
pass `Authentication` header with value: `Bearer <session_token>`

## Error Handling ❗

Here are the possible errors you might encounter while using this service:
- **INVALID_STATE:** This error occurs when provided state is not in the redis database. Might be due to XSS attack.
- **INTERNAL_SERVER_ERROR:** This error indicates that something unexpected happened on the server side. If you encounter this error, please reach out to the service administrator for assistance.
- **INVALID_SESSION:** This error occurs when the provided session ID is invalid or revoked. Please ensure that you are using a valid session ID for your requests.
- **UNAUTHENTICATED:** This error indicates that no `session_id` cookie has been passed with the request. To access protected routes, make sure to include the `session_id` cookie containing a valid session ID.

Feel free to ask any questions if you need further clarification or assistance with this service. Enjoy secure and reliable authentication! 🔒✨