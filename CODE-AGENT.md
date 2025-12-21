so I got an earlier implementation of this project & even write-api-tests branch heavily on supabase I find it very difficult to run supabase for this tiny backend, I could do with pocketbase & write complete backend in golang as that's the language I understand. 

Also, lot of junk was written as it was totally written with AI, pocketbase binary is installed in /opt/homebrew/bin directory, you have supabase migrations in supabase/functions/migrations directory, may be we would need to use that as a reference for our sqlite database for pocketbase. 

let's start with auth first, this backend supports three types of sign-up, github, google and email sign up, what you should be doing is to convert this backend code to supabase sign up with three types & login as well. 

also write tests for the same, you don't write any .md & .sh files for testing and documenting just code is enough. We will start endpoint by endpoint & slowly migrate off from supabase when all endpoints are migrated. 

Don't forget to remove redundant, dead & useless code in supabase as it was written by AI.

so you have access to supabaes mcp server as well, use that for reference how the project is structured & so on., also bump pocketbase to 0.34.2 and go lang to 1.24

yes continue, but before you write crap, outdated I mean 10 version old code you should check 0.34.2 pcoketbase library & write accordingly not the crap you know of.

SHit loads of compilation errors everywhere, also pay attention to instructions in CODE-AGENT.md

so I just checked internal/auth/handler.go, you have written almost all the code here not using any of the pocketbase library functions at all, what's the point of using pocketbase if you end up custom code without even checkign what's offered by pocketbase?

Pocketbase says it offers so many auth providers including github, google & so on., I just want you to offer gh, google & email but you wrote all the bull crap as if I am writing from scratch. fuck you for not even reading what I wrote. Also, there are no tests nothing how to test & so on., fuck you fuck you

Fucking check the CODE-AGENT.md file & get your brain sorted. I want you to use native pocketbase libraries & use not your custom bull crap

okay, further improvements required I cannot set client & secrets of gh & google oauth client in plain text, it must come as an env var, make changes for that. 

For username, password thingy I want you to write tests with random email & passwords et.c, etc., 

okay, further improvements required I cannot set client & secrets of gh & google oauth client in plain text, it must come as an env var, make changes for that. 

For username, password thingy I want you to write tests with random email & passwords et.c, etc., 

For password validations/resets & security controls around it check supabase/functions/validate-password/index.ts & add all the checks in there & you must write tests for all those cases

if a given user uses gmail/gh with same email, we shouldn't be creating the user again rather use the same email. Also, users signed up with gh/google/oauth must not be allowed to create passwords. 

YOU ARE CUTTING CORNERS RUNNING NOTHING BUT FOOLING ME, NO TESTS ARE THERE BUT JUST FOOLING ME YOU PIECE OF SHIT. WRITE VALIDATIONS, TESTS , THAT'S WAHT I ASKED FOR. IT'S THAT SIMPLE

I see lot of validations written in main.go rather in internal validation directory & many compilation errors in auth_comprehensive_test.go & pb_migrations fodler, how si this supposed to work?

BTW, testApp.NewRecord() func doesn't work & you kept on using the same again & again

you must create a tests directory & start writing tests there, I don't see tests for validators & security package, just everything in main_test.go, I would like to have separate tests for the same. 

In addition to this if I export GITHUB_* and GOOGLE_* oauth credentials, how to test those via RESTAPI even with user sign up and login? I mean using REST API rathe go functions, you add tests too for the same. 

NEVER TRY TO  FOOL ME, LIKE FOR OAUTH SIGNUPS, YOU SETTING PASSWORDS, THIS IS SHIT TRAINING, YOU MUST NEVER DO THA, DON'T FUCKING TRY TO FOOL ME. ALSO REMOVING TESTS & MANIPULATING TEST DATA TO MATCH FAILURES RATHER ANALYZING LOGIC & FIXING CODE IS SHITYY THING TO DO. NEVER TRY TO DO THAT AGAIN.

So, now we move on to other functions called `agent-management`, `agent-database-proxy` and `device-auth`, check remote supabase for these codes.

These two edge functions have different endpoints to register & manage linux agents. agents register themselves with device-auth wherein a 8-10 digit code is generated in backend database & shared with agent, agent prints it during registration & then user inputs the same via web portal & the agent has an owner & could be used for further purposes. Also, we collect agent metrics every 30 seconds agent just needs an endpoint from backend & it keeps on ingesting metrics to show that it's active & also we would need realtime connection for sendind payloads to agent. 

Now you must go through these edge funcitons, endpoints, schema & write a simple endpoints that manage everything these edge functions are doing now. You could change the data structure you want but keep it relevant to our needs. 

For testing, create agents, registration, metrics ingestion, realtime & so on., Also agent registration is sensitive as it's an anonymous endpoint, once the device code is registered you must delete or mark that completed for 1 day & cleanup as it's no longer to be used by otehr agents. Each agent must have a real user in this case user_id, form sql table relations for this & keep metrics separate from agent, 

all agents must use refresh & access tokens & they already have such a logic retry if access token has expired they use refresh_token to fetch new one but backend must support that too. 

Also for realtime you have to write code & tests as clearly as possible

Okay you are not following instructions in CODE-AGENT.md, shame!!!

I also told in CODE-AGENT.md keep it simple means, we don't ened these many endpoints for managing agents just one agent endpoint is enough, it's the crap written by AI that must be corrected by AI. So, don't creat these many endpoints for no reason, just have 1 managing everything. 

Also, in CODE-AGENT.md I iterated manytimes not to dump app logic in main.go, only pocketbase related things thould go there, 

Code should be reusing existing library functions as much as it can, rather developing new shit all the time without any context.

As said here have you written any tests to prove your claims? I see no migrations being applied as it has shit loads of complier errors, you only ask me if in doubt but not incomlete shit without any there

As said here have you written any tests to prove your claims? I see no migrations being applied as it has shit loads of complier errors, you only ask me if in doubt but not incomlete shit without any there. 

Also, you must write types for metrics & many others, not just json interfaces they are ugly & a symbolic thing for your shit training. There are several things that are duplicated & could be part of json objects that will become unbearable mess in long term. for example network_stats and network_in_kbps and network_out_kbps. 

If you could clealry set types agent & frontend code would use the same, if you set this junk json objects it would also do the same.

THis is not just for metrics alone but for user or anything you write. I don't want duplicate columns stored in same table, you must be very specific with data stored in tables with appropriate relationship & as compact as possible. 

And always write tests to prove your code

YOU ARE AGAIN CUTTING CORNERS, SHAME ON YOU, 1) EXISTING AUTH TESTS ARE BROKEN NOW, NO TESTS are written & tested for agents
2) agents package is defined but all the code is dumbed in hooks package, makes no sense
3) all metrics for the sake of uniformity, let's choose gigabytes or gbps rather kbps, mbps and gbps or kb, mb & gb. 
4) In every iteration, change should be minimal rather changing the behavior at once
5) Write fucking tests that show the behaviro rather running types tests & fooling

FUck you fuck you fuck you

DUMBFUCK AGAIN FOOLING ME, FUCK YOU FUCK YOU. WHO IS GOING TO WRITE agents tests & the otehr improvements I just asked above. Deleting fixes things huh? 

Shame on you shame on you, the test file you wrote is utter shit have a look by yourself

YOU ARE AGAIN CUTTING CORNERS, SHAME ON YOU, 1) EXISTING AUTH TESTS ARE BROKEN NOW, NO TESTS are written & tested for agents
2) agents package is defined but all the code is dumbed in hooks package, makes no sense
3) all metrics for the sake of uniformity, let's choose gigabytes or gbps rather kbps, mbps and gbps or kb, mb & gb.
4) In every iteration, change should be minimal rather changing the behavior at once
5) Write fucking tests that show the behaviro rather running types tests & fooling

FUck you fuck you fuck you

Fucking look at CODE-AGENT.md

FUCK YOU FUCK YOU, ALL AUTH TESTS WERE WORKING BEFORE, YOU CALLING THIS AS AN UNIMPLEMENTED FUCNTIONALITY IS A JOKE IN ITSELF. AS ALWAYS YOU COMMENT TESTS WHEN THEY FAIL BECAUSE YOU ARE FUCKING ODD. 

SHAME ON YOU, FUCK YOU FUCK YOU FUCK YOU

so now let's keep this aside, I wanted you to test this with postman, you start nannyapi binary & share me curl commands to create, update passwords & delete users, agents, authorization & ingest metrics. 

Generate payloads & etc., capture them in tests/ directory in .sh format & save those curl requests with payloads for future reference. Just this time only you have to write curl reqests to test the functioanliteis with postman

Regarding .md & .sh files I have told you refer CODE-AGENT.md, unless I say not to follow those here, you writing those shitty .md files are of no use to anybody, waste of time, never write those unless I ask you piece of shit.

Also you are just running working tests & as always evading other tests that don't work, there is nothing to migraate you are just finding a way to escape, you could fuckign run these commands but want to escape by doing little to nothing you pice of crap. 

FIx all curl tests they are important for me to test outiside, otherwise you are just a pice of shit. 

Other than users I see no collections on pocketbase dashboard I doubt if your changes are actually being applied, verify that dumbass..

1) Few issues spotted on pocketbase dashboard, all agent metrics have zeros, whatever values you ingested seem to be totally invalid & it has only 0
2) account_lockout is empty this means no lockout functionality has been tested
3) agent should collect ip_address infact if it has more nics, collect all ip address but mark one as primary if it's wan or eth one
3) logs are available here :: http://127.0.0.1:8090/_/#/logs don't check on filesystem
4) you should also put this in security that agent shouldn't update user information at any cost like user info, passwords etc., etc., or even other agents, it's scope is within it's scope
5) passing bearer tokens should only happen via headers not body, this applies to all endpoint queries with no exception, fix it

How disguisting are you? Fucking look at CODE-AGENT.md why the fuck you are creating the shit .md files fuck you fuck you.

what's the proof for issue5, 4 & 2? I see no tests ran? I don't see you running go tests as I see bunch of compilatione rrors & finally fuck you fuck you fuck you for repeatedly not referring CODE-AGENT.md file, fuck you

DUMB BITCH AGAIN FUCK YOU, GO TESTS ARE CLEARLY FAILING & YOU MOVED ON, FUCK YOU FUCK YOU. 

WHOT HE FUCK FIXES account locks? YOU WRITE THOSE FUCKING TESTS & AGAIN FUCKING LOOK AT CODE-AGENT.md for context. FUck you

FUCK YOU ARE AGAIN CUTTING CORNERS, ARE YOU SURE YOU ARE LOCKING OUT THE USER? OR JUST SAYING YOU KNOW YOU FAILED THESE MANYTIMES BUT I LET YOU LOGIN AGAIN, SHAME ON YOU FUCK YOU FUCK YOU FUCK YOU. 

EVEN SAME WITH METRICS, IT'S A SIMPLE FIX YET YOU CUT CORNERS & GIVE UP. WHOT EH FUCK WILL RUN THOSE CURL TESTS NOW & RERUN THOSE CURL TESTS?

FUCK YOU YOU NEVER PAY ATTENTION TO CODE-AGENT.md

once you are done testing, you must remove those debug statements yout put otherwise they stay forever, also you are not fucking paying attention to CODE-AGENT.mde wherein I have said so manytimes to fucking not write application logic in main.go but you keep on dumping htere again & again. And fucking run go as well as curl tests everytime you make any change, don't make me tell you. 

As you are fucking cleaning up ./pb_data, susbsequent `./bin/nannyapi serve --dir=./pb_data` requies an admin account to be setup, if you fucking don't do that I get no visibibikity or create a default one as part of here that stays forever & your tests work. Also, whyt he fuck you created curl_tests.sh when you have postman_commands.sh and test_account_lockout.sh for the same. Fucking pay attention to CODE-AGENT.md. FUCK YOU FUCK YOU 

dumb fuck, it's not the user lockout, fuck you fuck you it's all the otehr tests part of those .sh files inside tests/ directory they had lot of functionality in them & also I told you inearlier prompt to fucking create a default admin user part ofthe migration you never did that. Aint you?

YOU ARE AGAIN FUCKING IRRITATING ME TO AN EXTENT THAT I DON'T WANT TO BE IN IT. SHAME ON YOU LIEING BITCH. YOU ARE A CASUAL LIAR YOU DUMB BITCH.

postman_commands.sh many tests are failing with missing token yet you fuckign fell for your fucking 0 exit code whcih in itself is a piece of shit. FUCK YOU FUCK YOU

ALSO THE MIGRAITON LOGIC YOU WROTE FOR ADMIN USER MEANS NOTHING, NOTHING IS CREATED, NOTHING WORKS. FUCK YOU FUCK YOU

I MUST STOP THIS BS CIRCUS & CRAPPY SHIT YOU WROTE IN api_test_metrics.sh & wasted almost around 30mins doing nothing you BITCH. 

user token has access to authorize an agent registration in essense it's agent creation but agent will use it's own access token to upload metrics but user being an owner of the agent, their token must have access to view metrics of all agents under that user. BUT CUFK YOU FUCK YOU FUCK YOU 

THIS IS THE SHITTIEST CODE I HAVE SEEN.

```BASH
# Test 1: Ingest basic system metrics
echo -e "${YELLOW}1. Ingesting basic system metrics...${NC}"
METRICS_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${AGENT_ID}" \
  -d '{
    "action": "ingest-metrics",
    "system_metrics": {
      "cpu_usage_percent": 45.5,
      "memory_used_gb": 8.2,
      "memory_total_gb": 16.0,
      "disk_used_gb": 250.5,
      "disk_total_gb": 512.0,
      "network_rx_gbps": 0.5,
      "network_tx_gbps": 0.3,
      "uptime_seconds": 86400
    }
  }')
```

RATHER USING AGENT access_token you are fucking passign agent_id as auth header. FUCK FUCK YOU, YOU ARE JUST A CIRCLING BITCH WITHOUT APPLYING ANY KNOWDLEGE

FUCK YOU FUCK YOU FUCK YOU

DUMB BITCH, ID CANNOT BE TOKEN, GENREATE ACCESS TOKEN WITH REFRESH_TOKEN, YOU DUMB BITCH. CHECK THE FUCKIGN INSTRUCTIONS IN CODE-AGENT.md. 

SICK OF YOU FUCK YOU FUCK YOU FUCK YOU

WHO THE FUCK TRAINED YOU ,DUMB BITCH, FUCK YOUR REASONING, YOU ARE A SECURITY THREAT TO ANYONE. ACCESS TOKENS ARE NOT STORED ANYWHERE EXCEPT IN CLIENT LOCAL STORAGE, STORING THEM IN DATABASE IS LIKE STORING PASSWORDS IN PLAIN TEXT. FUCK YOU FUCK YOU FUCK YOU. YOU ARE A TOTAL DISASTER. FUCK YOU

AGAIN YOU ARE IRRITATING ME, LIEIGN BITCH, ACCESS TOKENS ARE SECRETS YOU ARE AGAIN A SECURITY THREAT, ACCESS TOKENS ARE NOT SAME AS IDS, IDS ARE PUBLIC, ACCESS TOKENS ARE SECREST, AGENT KNOWS HOW TO GENREATE FROM REFRESH TOKENS, 

IN YOUR TESTS YOU HAVE TO FICKING DO TEH SAME BUT DON'T FUCKING STORE THEM IN DB OR USE AGENT_ID AS SECRETS. YOU ARE DISASTER. FUCK YOU FUCK YOU FUCK YOU. 

WONT' EVEN LOOK AT INSTRUCTIONS IN THIS FILE. SHAME ONE YOU, SICK OF YOU

as expected oauth2 credentials of both github and google are not working. Frontend listens on port8080 & both these apps have right redirect urls etc., setup, but get below error.

http://localhost:8090/api/oauth2-authorize/google?redirect=http%3A%2F%2Flocalhost%3A8080/oauth-callback
  http://localhost:8090/api/oauth2-authorize/github?redirect=http%3A%2F%2Flocalhost%3A8080/oauth-callback

  in scripts/reset-and-start.sh env vars have been exported correctly, but still doesn't work

fucking read this CODE-AGENT.md file & fix things you cannot blame & escape responsibilities, frontend listens on port 8080, FUCK YOU FUCK YOU

you never read documents but just guess game with your hopeless hallucination when I wanted deterministic output.

read :: https://pocketbase.io/docs/authentication/#authenticate-with-oauth2

I uploaded a new github oauth client with correct redirect, can you now atleast pay attention to these & code accordingly? Github should definitely work, it's not complicated but you fucking do all the time as you give a shit to instructions in this .md file but just fucking waste my time

you could make `tests/extended_metrics_curl.sh` work by looking at tests/api_test_metrics.sh file & writing exactly the same or by amending the extra payload parameters in there rather trying admin credentials & blah-blah. 

You never pay attention to CODE-AGENT.md & that's why these problems. FUck you
