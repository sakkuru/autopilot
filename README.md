# Blue Green Push / Rollback

This is a Cloud Foundry CLI plugin. I learn/forked code from the great plugin, [autopilot](https://github.com/contraband/autopilot). This feature enable you to zero-downtime-deploy with versioning and Rollback feature. 

# install 

```
cf insall-plugin PATH_TO_BLUE_GREEN_BINARY
``` 

# 1. deployment

`blue-green-push` command enable you to deploy with versioning.

## 1.1. usage

```
cf blue-green-push application-name -f manifest.yml
```

## 1.2. behavior

The command push your application. If the application has old version,
the command rename the current version to `applicationName-g1` and unmap
gracefully and stop it. This command save two versions.

The command behaves like this.

```
cf rename application-to-replace application-to-replace-g1
cf push -f path/to/new_manifest.yml -n application-to-replace-staging
cf map-route application-to-replace-staging xxxx.io (domain name) -n application-to-replace
cf unmap-route application-to-replace-g1 xxxx.io (domain name) -n application-to-replace
cf stop application-to-replace-g1
```

# 2. Rollback

`blue-green-rollback` command rollback to suitable version.

## 2.1. usage

```
cf blue-green-rollback application-name version (e.g. g1 or g2)
```

## 2.2. behavior

The command rollback your application. Start the old version and map route
the host which is current url. Then unmap the current version. After that,
it swap the application name. 

This is the similar behavior of this rollback feature.

```
cf start application-name-g1
cf map-route application-name-g1 xxxx.io (domain name) -n application-name
cf unmap-route application-name xxxx.io (domain name) -n application-name
cf rename application-name application-name-now-on-swapping
cf rename application-name-g1 application-name
cf rename application-name-now-on-swapping application-name-g1
```

# 3. Versioning

This tool leave maximum three versions for rollback.

```
application-to-replace (current)
application-to-replace-g1
application-to-replace-g2
```

If you deploy the new version, g2 will be removed. then current -> g1 and g1 -> g2.

# 4. Pland feature

Now we support push/rollback feature. We will develop rolling update feature and
specify the route feature if the app have mutiple domain.  

Enjoy coding!



