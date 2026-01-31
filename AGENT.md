1) Now, I want API to support sbom scanning ie., agents will send archived format of sbom, I am thinking of uisng `syft` on client machines, it ingests both host and podman container related vulnerabilities using sbom, API will extract the sbom files, finds vulnerabilities using grype or equivalent library & show the results back to client.

2) Delete the sbom file once it's parsed. Make sure you extract the right ones & put appropriate security controls in place

3) Store results in database so that you could show vulnerabilities per agent, store in an efficient way as more agents would totally complicate the scenario

4) Make this optional by configuring a cli argument, if that's not set no need to do vulnerability management, but if agents send something or frontend you just return an error message saying it's not configured but provide them an option to activate

5) Change the install.sh & docker images to install necessary grype binary or library whatever you find reasonable

6) Update docs/ page with this info & as usual tests & lint checks must be run, test exhaustively this usecase

7) I am not sure of what's the right table structure, sbom/grype/syft commands etc., etc., but you could install on this mac & find out those for proper exchange of data

8) Document all of this in .md files for agent & frontend

9) Write a cron to download the grype db files so that scanning could be faster

Docs for grype :: https://github.com/anchore/grype/
