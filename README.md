# YaConTable — generate and join tables for Yandex contests

## Building

This is just a simple project on Golang, so no complex steps are required. Just do

```
$ go build -o yacontable ./cmd
```

to build the executable.

## Configuration

For configuration, you will need to create two files.

The first one must be called `config.json`. See [internal/config.go](internal/config.go) for all the available options.

For example, the config may look as follows:

```json
{
    "listen_addr": "0.0.0.0:80",
    "secure_listen_addr": "0.0.0.0:443",
    "allowed_secure_domains": ["yacontable.example.com"],
    "base_url": "https://yacontable.example.com",
    "contests": [
        {
            "id": 123456,
            "tag": "Day1"
        },
        {
            "id": 123457,
            "tag": "Day2"
        },
        {
            "id": 123458,
            "tag": "Day3"
        },
        {
            "id": 123459,
            "tag": "Day4"
        }
    ],
    "login_whitelist_regex": "^my-login-",
    "login_blacklist_regex": "^my-login-(42|43|44)$",
    "max_score_per_task": 100.0
}
```

The second one is `secrets/static.json`. It is needed to interact with Yandex Contest API.

You have to visit [this link](https://oauth.yandex.ru/client/new/) to create an application. While creating the application, do not forget the following:
- The application must have `contest:submit` and `contest:manage` scopes.
- Add your redirect URL properly! For example, if your base URL is `https://yacontable.example.com`, then your redirect URL must be `https://yacontable.example.com/authCallback`. By default, the base URL is `http://localhost:8080`, so for testing purposes you must set your redirect URL to `http://localhost:8080/authCallback`.

After creating your application, put the following into `secrets/static.json`:

```json
{
    "client_id": "YOUR CLIENT ID",
    "client_secret": "YOUR CLIENT SECRET"
}
```

When running the server for the first time, you must log in to Yandex Contest API. All you have to do is to find an login URL in the logs, open it and grant access using this link. After this procedure, you allow the server to use Yandex Contest API on your behalf.

## License

The project is distributed under the terms of MIT License. See [LICENSE](LICENSE) for more details.
