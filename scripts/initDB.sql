CREATE TABLE `UserInfo` (
	`UserID` VARCHAR(64)  PRIMARY KEY,
	`Username` VARCHAR(64) NULL,
	`Password` VARCHAR(64) NULL,
	`TwitterUsername` VARCHAR(64) NULL,
	`TwitterAuthToken` VARCHAR(64) NULL,
	`TwitterSecret` VARCHAR(64) NULL,
	`RedditUsername` VARCHAR(64) NULL,
	`RedditAuthToken` VARCHAR(64) NULL,
	`FacebookUsername` VARCHAR(64) NULL,
	`FacebookAuthToken` VARCHAR(64) NULL,
	`RedditWeight` INTEGER,
	`FacebookWeight` INTEGER,
	`HackerNewsWeight` INTEGER,
	`GoogleNewsWeight` INTEGER,
	`TwitterWeight` INTEGER
);
