### Withdrawal Autoclaim

```sh
docker build . -t autoclaimer 
```

Its necessary to attach checkpoint folder from host fs to container to write synced checkpoints and read last synced checkpoint once application starts:
```sh
docker run --env-file=.env -v  "$(pwd)/checkpoint:/app/checkpoint"  autoclaimer
```
