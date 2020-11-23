# MicroFileServer
Service for storing small files


## Configuration

File ```config.json``` must contain next content:

```js
{
  "DbOptions": {
    "uri": "mongodb://user:password@localhost:27017/MicroFileServer", //uri connection string | env: MFS_MONGO_URI
  },
  "AppOptions": {
    "testMode": true|false, //bool option for enabling Tests mode | env: MFS_TEST_MODE
    "appPort": "8080", //app port | env: MFS_APP_PORT
    "maxFileSize": 100, //maximum file size for upload in MB | env: MFS_MAX_FILE_SIZE
    "pathPrefix": "/example"    //URL path prefix | env: MFS_PATH_PREFIX
  }
}
```

File ```auth_config.json``` must contain next content:

```js
{
  "AuthOptions": {
    "keyUrl": "https://examplesite/files/jwks.json", //url to jwks.json | env: MFS_AUTH_KEY_URL
    "audience": "example_audience", //audince for JWT | env: MFS_AUTH_AUDIENCE
    "issuer" : "https://exampleissuersite.com" //issuer for JWT | env: MFS_AUTH_ISSUER
  }
}

```

