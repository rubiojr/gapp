Added a new public [glance package](/pkg/glance) exposing a `ServeApp` function.

The original glance package is vendored in internal/glance.
I had to make 3 functions public, for them to be used in the new public package.

* application.Server
* NewApplication
* NewConfigFromYAML

The public ServeApp:

* Doesn't watch the config file, as it won't be necessarily hosted by the filesystem (a text blob, an binary embedded file, etc).
* Receives a context, backgrounds the HTTP server
* Takes a string as the configuration, instead of a filesytem file
* Takes a number of options, to configure the Glance server

Whislist:

* Glance could expose a UNIX socket instead of a TCP one
* The ability to define the config programatically could be interesting (maybe already possible)
