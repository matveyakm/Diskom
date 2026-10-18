-- Late insert (executed by build-db.sh after all entities exist):
-- acceptance_document's related document is appointment_document.
INSERT INTO related_document(system_object, document, output_order)
SELECT (SELECT so.id FROM system_object so JOIN object_link ol ON ol.id = so.object_link WHERE ol.path = 'acceptance_document'),
       (SELECT so.id FROM system_object so JOIN object_link ol ON ol.id = so.object_link WHERE ol.path = 'appointment_document'),
       0;
