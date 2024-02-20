-- in cdb$root
alter session set current_schema = system;
create table init_done(t timestamp);
insert into init_done values(systimestamp);
commit;
exit;
