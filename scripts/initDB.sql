CREATE TABLE `UserInfo` (
	`UserID` VARCHAR(64)  PRIMARY KEY,
	`Username` VARCHAR(64) NULL,
	`Password` VARCHAR(64) NULL,
	`RedditUsername` VARCHAR(64) NULL,
	`RedditAuthToken` VARCHAR(64) NULL,
	`RedditTokenExpiry` TIMESTAMP NULL
);
