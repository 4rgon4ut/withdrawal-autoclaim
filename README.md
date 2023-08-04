### Withdrawal Autoclaim

```sh
docker build . -t autoclaimer
```

Its necessary to attach checkpoint folder from host fs to container to write synced checkpoints and read last synced checkpoint once application restarts.
Last synced block can also be provided as argument `--last-synced <desired_number>`. It should be passed when application starts fro the first time.
Start container:
```sh
docker run --env-file=.env -v  "$(pwd)/checkpoint:/app/checkpoint"  autoclaimer --last-synced <desired_number>
```
