print("Started Adding the Users.");
db = db.getSiblingDB("admin");
db.createUser({
    user:"admin",
    pwd: "password",
    roles: ["readWriteAnyDatabase"]
});
print("End Adding the User Roles.");