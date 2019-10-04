# Tasks

[![wercker status](https://app.wercker.com/status/980040d793775c6ebf128aacbde3ca76/m "wercker status")](https://app.wercker.com/project/bykey/980040d793775c6ebf128aacbde3ca76)

Contains:

* Receiver: the web server that (generally) receives http requests and turns them into task queue messages. This is the external interface to the tasks system.
* Runners: various task-pollers that perform miscellaneous functions. These should all eventually be moved into their own repos, since the only thing they share with one another is that their work is prompted by polled task messages rather than http request.
