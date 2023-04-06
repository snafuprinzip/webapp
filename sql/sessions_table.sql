CREATE TABLE IF NOT EXISTS `sessions` (
  `id` varchar(255) NOT NULL DEFAULT '',
  `userid` varchar(255) NOT NULL DEFAULT '',
  `expiry` timestamp NOT NULL DEFAULT current_timestamp() ON UPDATE current_timestamp(),
  PRIMARY KEY (`id`),
  KEY `user_id_idx` (`userid`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
