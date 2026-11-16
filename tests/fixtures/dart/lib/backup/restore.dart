import 'dart:io';

import 'package:archive/archive.dart';
import 'package:file_picker/file_picker.dart';
import 'package:path_provider/path_provider.dart';

class BackupRestore {
  static Future<int> importFromFile() async {
    final picked = await FilePicker.platform.pickFiles(
      type: FileType.custom,
      allowedExtensions: ['zip'],
      withData: true,
    );
    final bytes = picked?.files.single.bytes;
    if (bytes == null) return 0;

    final dir = await getApplicationDocumentsDirectory();
    final archive = ZipDecoder().decodeBytes(bytes);
    var count = 0;
    for (final entry in archive.files) {
      if (!entry.isFile) continue;
      final out = File('${dir.path}/backup/${entry.name}');
      await out.create(recursive: true);
      await out.writeAsBytes(entry.content as List<int>);
      count++;
    }
    return count;
  }
}
