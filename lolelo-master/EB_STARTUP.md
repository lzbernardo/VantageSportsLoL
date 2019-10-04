UPDATING
=====

EloBuddy starts up via the ElobuddyLoader.exe, which attempts to update itself using the following process.
1) Makes request for https://raw.githubusercontent.com/EloBuddy/EloBuddy.Dependencies/master/dependencies.json
2) Compares the MD5 of the listed dependencies with those in its ProgramFiles folder and Users\AppData\Roaming\EloBuddy\Addons\Libraries folder.
3) For any that are different, downloads the updated version using the json path.

It appears it just started using this dependencies.json method as of September 8, 2016.

If we want to manipulate dependencies, we could probably modify C:\Windows\System32\drivers\etc\hosts to point raw.githubusercontent.com at a server we control, and respond to requests for files with the copies we want used.

LOGIN
====
If you modify the hosts file on the machine to point auth.elobuddy.net at a bogus IP, EB will login as guest with no obvious side-effects. Useful for preventing EB from knowing that we're logging in!