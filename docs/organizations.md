# Organizations

Organizations can:

* sign users' pubkey
* revoke user's pubkey
* sign sub organization's pubkeys

you can:

* list an organization's users
* add an user through that organization

example:

1. I add davidw through nccgroup's organization
2. I now see davidw@nccgroup in my contact list

1. I add masonh through nccgroup's cryptoservices suborganization
2. I now see masonh@nccgroup.cryptoservices in my contact list

# NO

- actually, having suborganizations is not going to be flexible, what if people move around?
- better might be to just have the organization organize users as it wishes (but this becomes a threat?)
- they can publish a change of user's sub-organization in the gossip merkle tree?
