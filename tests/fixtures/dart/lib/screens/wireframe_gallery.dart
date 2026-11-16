import 'package:flutter/material.dart';

class WireframeGallery extends StatelessWidget {
  const WireframeGallery({super.key});

  static const _screens = ['Đăng nhập', 'Danh sách giao dịch', 'Chi tiết giao dịch', 'Tạo QR'];

  @override
  Widget build(BuildContext context) => Scaffold(
        appBar: AppBar(title: const Text('Wireframes')),
        body: ListView(
          children: [for (final s in _screens) ListTile(title: Text(s))],
        ),
      );
}
