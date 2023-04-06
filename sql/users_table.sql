CREATE TABLE IF NOT EXISTS `users` (
  `id` varchar(255) NOT NULL DEFAULT '',
  `username` varchar(255) NOT NULL DEFAULT '',
  `email` varchar(255) NOT NULL DEFAULT '',
  `password` text NOT NULL,
  PRIMARY KEY (`id`),
  KEY `username_id_idx` (`username`),
  KEY `email_id_idx` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
