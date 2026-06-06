## Description
notifier is a standalone service that dispatches email to recipients on behalf of the server.
It is an program that will be used to forward mails and push notifications to users.
What problem does this solve?
Enterprises othen have to setup a mail server to notify user from some services. 
For example to send a mail to a user the automated process (IP) should be authorized in order to let him send report or notification mail.

Notifier unifies this by providing a single source of truth for all the incoming process. 
Instead of talking directly to the server, a request can be sent to Notifier and it will handle the rest. 
One single source of truth for all the others services. 

## authentication (apikey based)
Authentication is done by providing a api_key to the consumer after creation. 
A consumer will have an attributed sender email (eg:automater-noreply@domain.com) that is unique for the process it represents.

## protection and audit

Notifier should mandatory implement a mechanism of abuse-detection, anti-spam, Ddos.
Notifier must implement Rate-limit and job status to the consumer. 


