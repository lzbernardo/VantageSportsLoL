# lolvideo
Repo for generating videos from league replays

This contains a lot of instructions and code required to set up the video generation pipeline.

## Worker
The worker is an EC2, g2.2xlarge machine set up with the Windows-lol AMI.

This AMI contains all the setup, scripts, and applications required, but it's also nice to backup some of the stuff into a version control system.

The AMI contains:
* TightVNCServer that is start on instance startup.
* League client installed, and configured with display settings
* NVidia display drivers installed
* Sound driver installed and enabled (but vnc doesn't transmit audio, so this doesn't get us much)
* AWS Command Line Interface (CLI) installed with credentials (we don't use the AWS CLI anymore)
* Google cloud util Command Line Interface installed with credentials
* ffmpeg installed on Desktop

Provision worker machines through the coordinator. Go to /lolvideo/status and click the link to add workers

After provisioning, the worker will automatically query the Pubsub queue and issue requests to the coordinator to execute those requests. Upon successful completion, the message is acked and the process repeats.

## Coordinator
The coordinator is a dockerized webserver, whose purpose is to issue commands to the workers to record games.

The coordinator is required because the League Client will not run without a screen device. The coordinator is able to open up a virtual frame buffer and vnc into the machine to launch the client.

## Usage

### Start up the coordinator (do this first)

On desktops, use `docker-compose up`. Then, hit your localhost at port 9010 with /lolvideo/status

### Start up a worker

1. Go to /lolvideo/status
2. Click on the link to add workers
3. Enter the number of instances you want, and the maximum price per hour for each instance
4. After submitting, you have to wait for your spot requests to be fulfilled.
5. You don't need to do anything else. The machine will start up and automatically pull messages from the queue and contact the coordinator.
6. (Optional) You can log into the machine with vnc to debug it or see what's going on.
  * You need to get the dns name of the machine. On the /status page, you should see the publicDnsName of each instance.
  * Run a vnc client (like xvnc4viewer). Enter in the dns name and the password (ask kevin.lee@ for this)
  * You should be logged in. Don't leave it on for too long because it uses a lot of bandwidth, which isn't free.

### Testing it all works
1. Grab a replay (right now, it's set up for lolking.net. See constants.go in the coordinator)
2. Get a gameID, encryptionKey, platformID, and convert the gameLength into seconds. I usually add about 10 seconds to the length for buffering.
3. Take a look at worker/sampleMessage. Replace it with the appropriate values.
4. Go to the Google Pub/Sub console: https://console.cloud.google.com/cloudpubsub/topicList?project=vs-dev
5. Find the queue: lol-videogen-tasks. Next to it, should be a Publish button
6. Paste in the sampleMessage json into the text field and hit publish.
7. And we're off!
8. The best thing to monitor is the coordinator logs. This will contain all the activity.
9. You can also log into the worker machine to see what it's recording.
10. After it's done, there should be a file in gcs. Use `gsutil ls gs://vsp-esports/lol/video/` and there should be a file with the name "<platform>-<game_id>-<champFocus>.mp4". You can download this to verify its integrity.
11. The queue item should also be acknowledged. You can use the /status page to see that the queue size is lower. Note that the value takes about 3 minutes to update, so it's not immediate.

### Upgrading the image

1. Log into the amazon ec2 console. 
2. Find the AMI you want to upgrade from (typically the most recent one)
3. Launch an image using that AMI, making sure to use the default storage options
4. After a few minutes when that instance is ready and initialized, use the aws console to get the password and RDP config. Remote into the new machine.
5. Edit the aws config file C:\Program Files\Amazon\Ec2ConfigService\Settings\config.xml and change the `Ec2SetPassword` to Enabled and `Ec2HandleUserData` to Enabled.
6. Run the LolClient and allow the client upgrade to complete.
7. Open up EloBuddy and make sure that it downloads the latest compatible version.
8. Use IE to go to replay.gg and download a batch file for a recent game, and run it to make sure everything is working properly, and the events you expect are in the resulting txt file.
9. Once satisfied, go back into the AWS Console, right click on the running instance, and create a new image off of it.
10. Name the new image Windows-lol-PATCH_NUM. E.g. Windows-lol-6-19. Make sure the description includes the major and minor version of the patch number. E.g. "6.19.".
11. Once the AMI is ready, copy it to any other region you want to launch it from using the AWS console. You can now terminate the instance you created.
