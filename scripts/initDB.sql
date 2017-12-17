CREATE TABLE `UserInfo` (
	`UserID` VARCHAR(64)  PRIMARY KEY,
	`Username` VARCHAR(64) NOT NULL,
	`Password` VARCHAR(64) NOT NULL,
	`TwitterUsername` VARCHAR(64) NOT NULL DEFAULT "",
	`TwitterAuthToken` VARCHAR(64) NOT NULL DEFAULT "",
	`TwitterSecret` VARCHAR(64) NOT NULL DEFAULT "",
	`RedditUsername` VARCHAR(64) NOT NULL DEFAULT "",
	`RedditAuthToken` VARCHAR(64) NOT NULL DEFAULT "",
	`FacebookUsername` VARCHAR(64) NOT NULL DEFAULT "",
	`FacebookAuthToken` VARCHAR(64) NOT NULL DEFAULT "",
	`RedditWeight` FLOAT NOT NULL DEFAULT 0,
	`FacebookWeight` FLOAT NOT NULL DEFAULT 0,
	`HackerNewsWeight` FLOAT NOT NULL DEFAULT 0,
	`GoogleNewsWeight` FLOAT NOT NULL DEFAULT 0,
	`TwitterWeight` FLOAT NOT NULL DEFAULT 0
);

CREATE TABLE `Rss` (
    `UserID` VARCHAR(64) NOT NULL,
    `Feeds` TEXT NOT NULL,
    `Weight` FLOAT NOT NULL,
    `Name` VARCHAR(64) NOT NULL
);
