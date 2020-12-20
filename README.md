# Nesthub

Nesthub is a bridge between Nest thermostats and Apple HomeKit.

## How Nesthub is different from HomeBridge

Pros: 

+ Written in pure Golang.
    + Easy cross compilation. Compile on your Mac and deploy on a Raspberry Pi!
    + No dependency to install on the target machine.
    + No JavaScript, npm, Node.JS, thank you very much!
+ Very small in size.
+ One single binary to deploy.
+ The program simply runs in the foreground. No background service to manage.
    + Although you could write your own systemd service/init script.
+ Uses the official Google Smart Device Management API. Not a single hack here.
    + Much easier setup than HomeBridge.

Cons: 

- Google SDM API requires a one-time fee of $5. Shame on you, Google!
- No fancy UI. Probably only for someone who knows how to work the command line.

## How to set it up

1. Go to https://console.nest.google.com/device-access to register for the SDM
   API. You need to pay a one-time fee of $5 to Google in this step.
2. Create a Google Cloud Platform project in Google Cloud Console.
3. Create an OAuth 2.0 client.
   3.1 Go to https://console.cloud.google.com/apis/credentials, click "CREATE
       CREDENTIALS" and choose OAuth Client ID.
   3.2 It is likely that you will need to first configure the consent screen.
       Select "External" and click "Create". Fill in App name. Fill in user
       support email and Developer contact email (at the end of the form) with
       your email address. "Save and continue" through the remaining steps.
       After "Back to dashboard", click "Publish".
   3.3 Go back to https://console.cloud.google.com/apis/credentials, click
       "CREATE CREDENTIALS" and choose OAuth Client ID.
   3.4 Choose "Web application" as the application type. Add
       "http://localhost:7979" as redirect URI at the bottom of the page.
   3.5 Copy and save the client ID and the client secret.
4. Create a Smart Device Management project.
   4.1 Go to https://console.nest.google.com/device-access and click Create.
   4.2 Fill in the project name and click Next.
   4.3 Fill in the OAuth client ID you got in step 3.5
   4.4 Enable Events, and click Create project.
5. Create a service account for the GCP project.
   5.1 Go to https://console.cloud.google.com/apis/credentials, click "CREATE
       CREDENTIALS" and choose Service account
   5.2 Choose a service account name you like, and click CREATE.
   5.3 Choose "Owner" as the role of the account. Click CONTINUE.
   5.4 Click "DONE".
   5.5 Click the three dots under "Actions", and Create key. Choose JSON.
   5.6 Save the key securely. It will be used later.
6. Create Pubsub subscription.
   6.1 Go to https://console.cloud.google.com, select your project. Click the
       shell button on the top right corner.
   6.2 Execute "gcloud pubsub subscriptions create homebridge-pubsub
       --topic=projects/sdm-prod/topics/<Project ID>". Here, <Project ID>
       is the SDM project ID shown in the Device Access Console. Go to
       https://console.nest.google.com/device-access to look it up.
7. Prepare the config file. Copy config_example.json to config.json.
   7.1 For "SDMProjectID", use the Project ID shown in the Device Access
       Console. Go to https://console.nest.google.com/device-access and choose
       the project you just created.
   7.2 For "GCPProjectID", use the Project ID shown in the Google Cloud Platform
       Console. Go to https://console.cloud.google.com and choose your project.
   7.3 For "OAuthClientID" and "OAuthClientSecret", use the ID and secret you
       obtained in step 3.5.
   7.4 For "ServiceAccountKey", set it to the path to the Service Account key
       file you downloaded in step 5.6.
   7.5 For "OAuthToken", set it to a path where you want to store the OAuth
       token. Note that the token will be obtained in the next step, so do not
       worry if you don't know what it is.
8. Finish OAuth authorization.
   8.1 Execute "nesthub -setup". You will be redirect to a Google login page.
   8.2 Login using the account associated with your Nest thermostat.
   8.3 Enable all access. Ignore all warnings about "this app is not verified".
       The warnings are there because we are using the sandbox mode of Google's
       Smart Device Managment API. Google wants to warn you that you are
       potentially giving unverified developer access to your device, but YOU
       are BOTH the "unverified developer" and the "user" AT THE SAME TIME here.
   8.4 After the web page is redirected and prompts you to go back to Terminal,
       switch back to Terminal.
   8.5 The app should be running now.
   8.6 Go to Home app on your iPhone. Click "+". Click "Add Accessory". Click
       "I Don't Have a Code or Cannot Scan". Wait for the bridge to appear, and
       use code "77887788" to pair.

## Highlights on the system design

+ Uses SDM pubsub event stream. No active polling of the SDM API.
    + Does not hit the ridiculously low API rate limit.
+ Device state query (e.g. check temperature) is entirely local. (Low latency.)

## Acknowledgements

This project uses hc for a pure-go implementation of the HomeKit Accessory
Protocol. hc is authored by Matthias Hochgatterer and other contributors.

